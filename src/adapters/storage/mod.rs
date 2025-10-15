pub mod storage;
pub use storage::{HistoryStorageList, RouteStorage};

#[cfg(test)]
use super::storage::storage::{HistoryItem, HistoryStorage, Item};

#[test]
fn test_history_storage_list_grow_mixed() {
    let mut history = HistoryStorageList::new();
    history.grow(Item::Call("call1".to_string()));
    history.grow(Item::Write("write1".to_string()));
    history.grow(Item::Invalidation("invalidation1".to_string()));
    let items = history.items();
    assert_eq!(items.len(), 3);

    for item in items {
        match item {
            HistoryItem::Args(set) => assert!(set.contains("call1")),
            HistoryItem::Write(val) => assert_eq!(val, "write1"),
            HistoryItem::Invalidation(val) => assert_eq!(val, "invalidation1"),
        }
    }
}

#[test]
fn test_route_storage_grow() {
    let mut routes = super::storage::RouteStorage::new(10);
    let mut deps = Vec::new();
    deps.push(Item::Call("dep1".to_string()));
    routes.grow("caller1".to_string(), "call_args1".to_string(), deps);
}
