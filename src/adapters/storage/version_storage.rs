use super::storage::{HistoryStorage, Item};
use std::collections::HashMap;

/// Stores the current version of every piece of data that can be a dependency.
/// The key is a dependency path string (e.g., "users/123"), and the value is the version number.
pub struct VersionStore {
    current_timestamp: u64,
    versions: HashMap<String, u64>,
}

impl VersionStore {
    fn new() -> Self {
        VersionStore {
            current_timestamp: 0,
            versions: HashMap::new(),
        }
    }

    /// Increments the version for a given dependency path.
    /// This should be called whenever a "Write" or "Invalidation" event occurs.
    // TODO: send invalidation event to other nodes
    fn invalidate(&mut self, path: String) {
        self.current_timestamp += 1;
        self.versions.insert(path, self.current_timestamp);
    }

    /// Retrieves the current version of a dependency.
    fn get_version(&self, path: &str) -> u64 {
        self.versions.get(path).unwrap_or(&0).clone()
    }
}

/// Main struct to manage the versioning-based caching system.
/// An alternative to `HistoryStorageList` that uses versioning instead of history scanning.
pub struct VersionedHistoryStorage {
    version_store: VersionStore,
    pending_calls: HashMap<String, u64>,
}

impl VersionedHistoryStorage {
    pub fn new() -> Self {
        VersionedHistoryStorage {
            version_store: VersionStore::new(),
            pending_calls: HashMap::new(),
        }
    }

    fn start_request(&mut self, call_key: String) {
        let start_timestamp = self.version_store.current_timestamp;
        self.pending_calls.insert(call_key, start_timestamp);
    }

    pub fn validate_call(&mut self, args: Item, route: Vec<String>) -> bool {
        if let Item::Call(call_key) = args {
            // Pop the request to ensure it's a one-time validation.
            let start_timestamp = match self.pending_calls.remove(&call_key) {
                Some(ts) => ts,
                None => return false, // The request doesn't exist. TODO: should we return true here or define a new policy?
            };

            for dep_path in route {
                let invalidation_timestamp = self.version_store.get_version(&dep_path);
                if invalidation_timestamp > start_timestamp {
                    // A dependency was changed after our request started. Invalidate.
                    return false;
                }
            }

            // No conflicts found. The operation is valid.
            true
        } else {
            true
        }
    }

    /// Invalidates a piece of data, which will cause any cached calls
    /// that depend on it to fail validation.
    fn invalidate(&mut self, path: String) {
        self.version_store.invalidate(path);
    }
}

impl HistoryStorage for VersionedHistoryStorage {
    fn grow(&mut self, key: Item) {
        match key {
            Item::Write(path) | Item::Invalidation(path) => {
                self.invalidate(path);
            }
            Item::Call(call_key) => self.start_request(call_key),
        }
    }
}
