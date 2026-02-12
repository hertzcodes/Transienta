use crate::{
    adapters::storage::version_storage::Item,
    app,
    config::{self, ManagerConfig},
    manager::Manager,
};
use std::collections::HashMap;

fn create_manager() -> Manager {
    #[allow(deprecated)]
    let app = app::App::mock_app(config::Config {
        manager: ManagerConfig {
            name: "test".to_string(),
            port: 1,
            shard_id: 0,
            neighbours: HashMap::new(),
        },
        cache: config::CacheConfig::Redis(config::RedisConfig {
            tls: false,
            port: 6379,
            db: 0,
            host: "localhost".to_string(),
            username: None,
            password: None,
        }),
    });
    Manager::new(app)
}

#[test]
fn test_validate_call_no_history() {
    let mut manager = create_manager();
    let result = manager.validate_call(
        Item::Request("c1".to_string()),
        Item::CallID("1".to_string()),
        vec!["r1".to_string()],
        "caller".to_string(),
    );
    assert_eq!(result, false);
}

#[test]
fn test_validate_call_with_history_valid() {
    let mut manager = create_manager();
    let history = manager.get_history();
    history.grow(Item::CallID("c1".to_string()));
    let result = manager.validate_call(
        Item::Request("a1".to_string()),
        Item::CallID("c1".to_string()),
        vec!["r1".to_string()],
        "caller".to_string(),
    );
    assert_eq!(result, true);
}

#[test]
fn test_validate_call_same_request() {
    let mut manager = create_manager();
    let history = manager.get_history();
    history.grow(Item::CallID("c1".to_string()));
    history.grow(Item::Write("r1".to_string()));
    history.grow(Item::CallID("c2".to_string()));
    let result = manager.validate_call(
        Item::Request("a1".to_string()),
        Item::CallID("c1".to_string()),
        vec!["r1".to_string()],
        "caller".to_string(),
    );

    assert_eq!(result, false);

    let result = manager.validate_call(
        Item::Request("a1".to_string()),
        Item::CallID("c2".to_string()),
        vec!["r1".to_string()],
        "caller".to_string(),
    );

    assert_eq!(result, true)
}

#[test]
fn test_validate_call_with_history_invalid() {
    let mut manager = create_manager();
    let history = manager.get_history();
    // valid here
    history.grow(Item::CallID("c1".to_string()));
    // a write r1 happened this means c1 is no longer valid
    history.grow(Item::Write("r1".to_string()));
    history.grow(Item::Write("r2".to_string()));
    // history.grow(Item::Call("c1".to_string()));
    let result = manager.validate_call(
        Item::Request("a1".to_string()),
        Item::CallID("c1".to_string()),
        vec!["r1".to_string(), "r2".to_string()],
        "caller".to_string(),
    );
    assert_eq!(result, false);
}
