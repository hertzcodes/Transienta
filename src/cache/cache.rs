pub trait CacheProvider {
    fn set(&self, key: &str, value: &str);
    fn get(&self, key: &str);
}
