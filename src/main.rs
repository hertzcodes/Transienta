use transienta::config::Config;
use std::env;
fn main() {

    const DEFAULT_CONFIG_PATH: &str = "/etc/transienta/config.yaml";
    let conf_path = env::var("TRANSIENTA_CONFIG_PATH").unwrap_or(DEFAULT_CONFIG_PATH.to_string());

    let cfg = Config::new(conf_path);

}
