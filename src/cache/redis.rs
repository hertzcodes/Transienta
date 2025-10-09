use super::cache::CacheProvider;
use crate::config::{self, Config};
use redis::Commands;
pub struct Redis {
    connection: redis::Connection,
    config: redis::ConnectionInfo,
}

impl Redis {
    pub fn new(conf: &Config) -> Option<Self> {
        let cache_conf = match &conf.cache {
            config::CacheConfig::Redis(c) => c,
        };

        let config = redis::ConnectionInfo {
            addr: redis::ConnectionAddr::Tcp(cache_conf.host.clone(), cache_conf.port),
            redis: redis::RedisConnectionInfo {
                db: cache_conf.db,
                username: Some(cache_conf.username.clone()),
                password: Some(cache_conf.password.clone()),
                ..Default::default()
            },
        };

        let r = Redis {
            connection: Self::connect(&config),
            config: config,
        };

        return Some(r);
    }

    fn connect(conf: &redis::ConnectionInfo) -> redis::Connection {
        let client = redis::Client::open(conf.clone()).unwrap(); // FIXME
        client.get_connection().unwrap()
    }
}

impl CacheProvider for Redis {
    fn set(&self, key: &str, value: &str) {
        // self.connection.set(key, value);
        return;
    }

    fn get(&self, key: &str) {
        // self.connection.get(key);
        return;
    }
}
