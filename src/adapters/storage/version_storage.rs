use super::storage::{HistoryStorage, Item};
use std::collections::{HashMap, HashSet};
use std::rc::Rc;

/// Stores the current version of every piece of data that can be a dependency.
/// The key is a dependency path string (e.g., "users/123"), and the value is the version number.
pub struct VersionStore {
    current_timestamp: u64,
    versions: HashMap<Rc<String>, u64>,
    dependency_graph: HashMap<Rc<String>, Vec<Rc<String>>>, // map: dependency -> requests
    subscribers: HashMap<Rc<String>, Vec<Rc<String>>>,      // map: request -> caller
    referece_pool: HashSet<Rc<String>>, // this avoids duplicate copies of the same string
}

impl VersionStore {
    fn new() -> Self {
        VersionStore {
            current_timestamp: 0,
            versions: HashMap::new(),
            dependency_graph: HashMap::new(),
            subscribers: HashMap::new(),
            referece_pool: HashSet::new(),
        }
    }

    /// Increments the version for a given dependency path.
    /// This should be called whenever a "Write" or "Invalidation" event occurs.
    // TODO: send invalidation event to other nodes
    fn invalidate(&mut self, path: String) -> HashMap<Rc<String>, Vec<Rc<String>>> {
        self.current_timestamp += 1;
        let path_rc = self.get_or_create_rc(path);
        self.versions
            .insert(Rc::clone(&path_rc), self.current_timestamp);

        let mut invalidations = HashMap::new();
        if let Some(affected) = self.dependency_graph.remove(&path_rc) {
            for request in affected {
                if let Some(subscribers) = self.subscribers.remove(&request) {
                    invalidations.insert(Rc::clone(&request), subscribers);
                    self.referece_pool.remove(&request);
                    self.referece_pool.remove(&path_rc);
                    // we can let callers stay in reference pool since the maximum count is only the count of microservices
                }
            }
        }
        return invalidations;
    }

    /// Retrieves the current version of a dependency.
    fn get_version(&mut self, path: &str) -> u64 {
        // TODO: is it really necessary to create a key that is not present in versions inside reference pool?
        let rc = self.get_or_create_rc(path.to_string());
        self.versions.get(&rc).unwrap_or(&0).clone()
    }

    fn get_or_create_rc(&mut self, s: String) -> Rc<String> {
        if let Some(rc) = self.referece_pool.get(&s) {
            return Rc::clone(rc);
        }
        let rc = Rc::new(s);
        self.referece_pool.insert(Rc::clone(&rc));
        return rc;
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

    fn start_request(&mut self, request: String) {
        let start_timestamp = self.version_store.current_timestamp;
        self.pending_calls.insert(request, start_timestamp);
    }

    pub fn validate_call(&mut self, args: Item, route: Vec<String>, caller: String) -> bool {
        if let Item::Request(request) = args {
            // Pop the request to ensure it's a one-time validation.
            let start_timestamp = match self.pending_calls.remove(&request) {
                Some(ts) => ts,
                None => return false, // The request doesn't exist. TODO: should we return true here or define a new policy?
            };

            for dep_path in &route {
                let invalidation_timestamp = self.version_store.get_version(&dep_path);
                if invalidation_timestamp > start_timestamp {
                    // A dependency was changed after our request started. Invalidate.
                    return false;
                }
            }

            // save the request to the dependency graph
            let request_rc = self.version_store.get_or_create_rc(request);
            for dep_path in route {
                let dep_path_rc = self.version_store.get_or_create_rc(dep_path);
                if let Some(calls) = self.version_store.dependency_graph.get_mut(&dep_path_rc) {
                    calls.push(Rc::clone(&request_rc));
                } else {
                    let mut calls = Vec::new();
                    calls.push(Rc::clone(&request_rc));
                    self.version_store
                        .dependency_graph
                        .insert(dep_path_rc, calls);
                }
            }

            // save the caller to the subscribers map
            let caller_rc = self.version_store.get_or_create_rc(caller);
            if let Some(subscribers) = self.version_store.subscribers.get_mut(&request_rc) {
                subscribers.push(caller_rc);
            } else {
                let mut subscribers = Vec::new();
                subscribers.push(caller_rc);
                self.version_store
                    .subscribers
                    .insert(Rc::clone(&request_rc), subscribers);
            }

            // No conflicts found. The operation is valid.
            true
        } else {
            false // call doesn't exist
        }
    }

    /// Invalidates a piece of data, which will cause any cached calls
    /// that depend on it to fail validation.
    fn invalidate(&mut self, path: String) -> HashMap<Rc<String>, Vec<Rc<String>>> {
        return self.version_store.invalidate(path);
    }
}

impl HistoryStorage for VersionedHistoryStorage {
    fn grow(&mut self, key: Item) {
        match key {
            Item::Write(path) | Item::Invalidation(path) => {
                self.invalidate(path);
            }
            Item::Request(request) => self.start_request(request),
        }
    }
}
