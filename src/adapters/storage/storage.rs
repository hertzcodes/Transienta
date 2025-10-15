use lru;
use std::{
    collections::{HashMap, HashSet},
    num::NonZero,
};
pub trait HistoryStorage {
    fn grow(&mut self, key: Item);
}

// FIXME
type Call = String;
type Write = String;
type Invalidation = String;
type ServiceName = String;

#[derive(PartialEq, Eq, Hash)]
pub enum Item {
    Call(Call),
    Write(Write),
    Invalidation(Invalidation),
}

pub enum HistoryItem {
    Args(HashSet<Call>),
    Write(Write),
    Invalidation(Invalidation),
}

type CallersAndCalls = HashMap<ServiceName, Call>; // service_name, call

pub struct HistoryStorageList {
    items: Vec<HistoryItem>,
    call_indexes: HashMap<Call, Vec<usize>>,
}

#[deprecated(note = "use VersionedHistoryStorage instead")]
impl HistoryStorageList {
    pub fn new() -> Self {
        return HistoryStorageList {
            items: Vec::new(),
            call_indexes: HashMap::new(),
        };
    }

    pub fn items(&mut self) -> &mut [HistoryItem] {
        return &mut self.items;
    }

    pub fn get_items_and_locations(
        &mut self,
    ) -> (&mut Vec<HistoryItem>, &mut HashMap<Call, Vec<usize>>) {
        (&mut self.items, &mut self.call_indexes)
    }
}

impl HistoryStorage for HistoryStorageList {
    fn grow(&mut self, key: Item) {
        match key {
            Item::Call(data) => {
                // borrows the last item as mutable
                if let Some(HistoryItem::Args(last)) = self.items.last_mut() {
                    last.insert(data);
                } else {
                    // creates a hashset if it doesn't exist or it's not a HistoryItem of type Args
                    let mut other: HashSet<Call> = HashSet::new();
                    other.insert(data);
                    self.items.push(HistoryItem::Args(other));
                }
            }
            Item::Write(data) => self.items.push(HistoryItem::Write(data)),
            Item::Invalidation(data) => self.items.push(HistoryItem::Invalidation(data)),
        }
    }
}

pub struct RouteStorage {
    items: lru::LruCache<Item, CallersAndCalls>,
}

impl RouteStorage {
    pub fn new(size: usize) -> Self {
        // TODO: set call back on evictions
        let cache: lru::LruCache<Item, CallersAndCalls> =
            lru::LruCache::new(NonZero::new(size).unwrap());
        return RouteStorage { items: cache };
    }

    pub fn grow(&mut self, caller: String, call_args: String, deps: Vec<Item>) {
        for item in deps {
            if let Some(existing) = self.items.get_mut(&item) {
                existing.insert(caller.clone(), call_args.clone());
            } else {
                let mut new_map = CallersAndCalls::new();
                new_map.insert(caller.clone(), call_args.clone());
                self.items.push(item, new_map);
            }
        }
    }
}

pub struct HistoryStorageListWithDeps {
    pub history: HistoryStorageList,
    routes: RouteStorage,
}

impl HistoryStorageListWithDeps {
    pub fn new(routes_size: usize) -> Self {
        return HistoryStorageListWithDeps {
            history: HistoryStorageList::new(),
            routes: RouteStorage::new(routes_size),
        };
    }

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

    pub fn history(&mut self) -> &mut HistoryStorageList {
        return &mut self.history;
    }

    pub fn routes(&mut self) -> &mut RouteStorage {
        return &mut self.routes;
    }
}
