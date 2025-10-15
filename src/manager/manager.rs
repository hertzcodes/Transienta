use crate::adapters::storage;
use crate::adapters::storage::storage::{HistoryItem, Item};
use crate::{
    adapters::storage::{HistoryStorageList, RouteStorage},
    app,
};
use std::collections::HashSet;

#[derive(Debug)]
pub enum ManagerErrors {
    StartupFailure,
    ShutdownError(String),
}

pub struct Manager {
    app: app::App,
    history: storage::HistoryStorageList,
    routes: storage::RouteStorage,
}

impl Manager {
    pub fn new(a: app::App) -> Self {
        return Manager {
            app: a,
            history: HistoryStorageList::new(),
            routes: RouteStorage::new(1000),
        };
    }
    // TODO: this has to return an immutable reference or be deleted later
    pub fn get_history(&mut self) -> &mut HistoryStorageList {
        return &mut self.history;
    }

    // TODO: can be optimized
    pub fn validate_call(&mut self, args: Item, route: Vec<String>) -> bool {
        let route_set: HashSet<String> = route.into_iter().collect();

        for item in self.history.items().iter_mut().rev() {
            let validation = match item {
                HistoryItem::Args(set) => {
                    if let Item::Call(call_value) = &args {
                        if set.contains(call_value) {
                            set.remove(call_value);
                            return true;
                        }
                    }
                    true
                }
                HistoryItem::Write(data) => !route_set.contains(data),
                HistoryItem::Invalidation(data) => !route_set.contains(data),
            };

            if !validation {
                return false;
            }
        }
        // FIXME: this means history was empty
        return true;
    }
}
