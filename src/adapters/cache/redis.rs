use std::fmt::Debug;

use super::cache::CacheProvider;
use crate::config::{self, Config};
use redis::{Commands, FromRedisValue, RedisResult};
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
                username: cache_conf.username.clone(),
                password: cache_conf.password.clone(),
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
    fn set(&mut self, key: &str, value: &[u8]) -> RedisResult<()> {
        self.connection.set::<&str, &[u8], ()>(key, value)?;
        Ok(())
    }

    fn get(&self, key: &str) {
        // self.connection.get(key).ok();
        todo!("not implemented yet")
    }

    fn del(&mut self, key: &str) -> RedisResult<()> {
        self.connection.del::<&str, ()>(key)?;
        Ok(())
    }
}
