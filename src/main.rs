use dotenvy::dotenv;
use std::env;
use transienta::{app, config::Config, manager::manager};
fn main() {
    dotenv().ok();
    const DEFAULT_CONFIG_PATH: &str = "/etc/transienta/config.yaml";
    let conf_path = env::var("TRANSIENTA_CONFIG_PATH").unwrap_or(DEFAULT_CONFIG_PATH.to_string());

    let cfg = Config::new(conf_path);

    let app = app::App::new(cfg);

    let mut manager = manager::Manager::new(app);

    manager.run();
}
