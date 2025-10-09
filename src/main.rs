use dotenvy::dotenv;
use std::env;
use transienta::config::Config;
fn main() {
    dotenv().ok();
    const DEFAULT_CONFIG_PATH: &str = "/etc/transienta/config.yaml";
    let conf_path = env::var("TRANSIENTA_CONFIG_PATH").unwrap_or(DEFAULT_CONFIG_PATH.to_string());

    let cfg = Config::new(conf_path);
}
