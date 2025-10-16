use crate::adapters::storage::storage::Item;
use crate::{adapters::storage::version_storage, app};

#[derive(Debug)]
pub enum ManagerErrors {
    StartupFailure,
    ShutdownError(String),
}

pub struct Manager {
    app: app::App,
    storage: version_storage::VersionedHistoryStorage,
}

impl Manager {
    pub fn new(a: app::App) -> Self {
        return Manager {
            app: a,
            storage: version_storage::VersionedHistoryStorage::new(),
        };
    }
    // TODO: this has to return an immutable reference or be deleted later
    pub fn get_history(&mut self) -> &mut version_storage::VersionedHistoryStorage {
        return &mut self.storage;
    }

    pub fn validate_call(&mut self, args: Item, route: Vec<String>) -> bool {
        return self.storage.validate_call(args, route);
    }
}
