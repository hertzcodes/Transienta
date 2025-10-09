use crate::cache::{self, CacheProvider};
use crate::config;
pub struct App {
    cfg: config::Config,
    database: Option<Box<dyn CacheProvider>>,
}

impl App {
    pub fn new(cfg: config::Config) -> Self {
        let mut app = App {
            cfg: cfg,
            database: None,
        };

        app.set_db();

        return app;
    }

    fn set_db(&mut self) {
        self.database = match &self.cfg.cache {
            config::CacheConfig::Redis(_) => {
                cache::Redis::new(&self.cfg).map(|r| Box::new(r) as Box<dyn CacheProvider>)
            }
        }
    }

    pub fn config(&self) -> &config::Config {
        return &self.cfg;
    }

    pub fn cache(&mut self) -> &Box<dyn CacheProvider> {
        return match &self.database {
            Some(db) => db,
            None => panic!("no database found"), // TODO: handle the None value
        };
    }
}
