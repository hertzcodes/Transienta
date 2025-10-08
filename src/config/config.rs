use serde::{Deserialize, Serialize};
use serde_yaml;
use std::fs;

#[derive(Debug, Serialize, Deserialize)]
pub struct Config {
    pub service_name: String,
    pub cache: CacheConfig,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(tag = "type", content = "config")]
pub enum CacheConfig {
    #[serde(rename = "redis")]
    Redis(RedisConfig),
}

#[derive(Debug, Serialize, Deserialize)]
pub struct RedisConfig {
    #[serde(default)]
    pub ssl: bool, 
    #[serde(default = "default_redis_port")]
    pub port: u16,
    #[serde(default)]
    pub db: i64,
    #[serde(default = "default_redis_host")]
    pub host: String,
    #[serde(default)]
    pub username: String,
    #[serde(default)]
    pub password: String,
}

fn default_redis_host() -> String {
    return "127.0.0.1".to_string()
}


fn default_redis_port() -> u16 {
    return 6379
}

impl Config {
    pub fn new(path: String) -> Self {
        let content = fs::read_to_string(&path).expect(&format!("couldn't find file in {}", &path));

        let config: Config = serde_yaml::from_str(&content).expect(&format!("couldn't parse config in {}", &path));

        return config;
    }
}