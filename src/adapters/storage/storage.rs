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
#[derive(PartialEq, Eq, Hash, Clone)]
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
}

impl HistoryStorageList {
    pub fn new() -> Self {
        return HistoryStorageList { items: Vec::new() };
    }

    pub fn items(&self) -> &Vec<HistoryItem> {
        return &self.items;
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
