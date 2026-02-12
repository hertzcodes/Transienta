use crate::adapters::comms::fbs;
use crate::adapters::comms::fbs::RequestOffset;
use crate::adapters::comms::nanomsg;
use crate::adapters::storage::version_storage::Item;
use crate::{adapters::storage::version_storage, app};
use nng::Message;
use nng::options::protocol::pair;
use nng::{Error, Protocol, Socket};
use redis::ToRedisArgs;
use std::sync::mpsc;
use std::{env, process, str, thread, time::Duration};

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

    pub fn run(&mut self) {
        // let bus_sock = nanomsg::connect_bus(self.app.config(), &[""]).unwrap();
        let (tx, rx) = mpsc::channel();
        let manager_config = self.app.config().manager.clone();
        let wrapper_thread = thread::spawn(move || {
            let pair_sock = nanomsg::connect_pair(&manager_config).unwrap();
            loop {
                let message = pair_sock.recv();
                if let Ok(msg) = message {
                    tx.send(msg);
                } else {
                    // log error
                }
            }
        });

        // TODO: request processing can be done zero copy with lifetimes later
        for message in rx {
            let root = fbs::root_as_request(&message);
            if let Ok(data) = root {
                match data.request_type() {
                    fbs::RequestUnion::StartRequest => {
                        let request = data.request_as_start_request().unwrap();
                        self.storage
                            .grow(Item::CallID(request.id().unwrap().to_owned().to_string()));
                    }
                    fbs::RequestUnion::EndRequest => {
                        let request = data.request_as_end_request().unwrap();
                        let (args, id, caller) = (request.args(), request.id(), request.caller());
                        let deps = request
                            .deps()
                            .map(|vec| vec.iter().map(|s| s.to_string()).collect())
                            .unwrap_or_else(Vec::new);
                        if self.storage.validate_call(
                            Item::Request(args.to_string()),
                            Item::CallID(id.to_owned().unwrap().to_string()),
                            deps,
                            caller.to_owned().unwrap().to_string(),
                        ) {
                            // send save request to the caller
                        }
                    }
                    fbs::RequestUnion::InvalidationRequest => {
                        let request = data.request_as_invalidation_request().unwrap();
                        let affected = self
                            .storage
                            .invalidate(request.key().to_owned().unwrap().to_string());
                        // send affected calls as ManagerInvalidateRequest
                    }
                    fbs::RequestUnion::ManagerInvalidateRequest => (),
                    fbs::RequestUnion::ManagerSaveRequest => (),
                    _ => (), // ignore since it's an invalid request
                }
            }
        }
    }

    #[cfg(test)]
    pub fn get_history(&mut self) -> &mut version_storage::VersionedHistoryStorage {
        return &mut self.storage;
    }

    #[cfg(test)]
    pub fn validate_call(
        &mut self,
        args: Item,
        call_id: Item,
        route: Vec<String>,
        caller: String,
    ) -> bool {
        return self.storage.validate_call(args, call_id, route, caller);
    }
}
