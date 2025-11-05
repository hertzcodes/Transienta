use redis::RedisResult;

pub trait CacheProvider: Send + Sync {
    fn set(&mut self, key: &str, value: &[u8]) -> RedisResult<()>;
    fn get(&self, key: &str); // TODO: is it even useful?
    fn del(&mut self, key: &str) -> RedisResult<()>;
}
