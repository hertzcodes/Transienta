use super::requests_generated::fbs;
use crate::adapters::comms::fbs::root_as_request;
use crate::adapters::comms::requests_generated::fbs::RequestUnion;
use crate::config::{self, ManagerConfig};
use flatbuffers::{self, InvalidFlatbuffer};
use nng::{Error, Protocol, Socket};
use std::{env, process, str, thread, time::Duration};

pub struct NNGHandler {
    pub bus_sock: Socket,
    pub pair_sock: Socket,
}

fn connect_bus(config: &config::ManagerConfig, peers: &[String]) -> Result<Socket, Error> {
    let s = nng::Socket::new(Protocol::Bus0)?;
    // s.listen(format!("tcp://{}:{}", config.name,config.port))?;
    for peer in peers {
        s.dial(peer);
    }
    Ok(s)
}

pub fn connect_pair(config: &config::ManagerConfig) -> Result<Socket, Error> {
    let s = nng::Socket::new(Protocol::Pair0)?;
    s.listen("tcp://127.0.0.1:5532")?;
    Ok(s)
}
