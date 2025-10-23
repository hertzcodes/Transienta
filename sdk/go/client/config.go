package client

type ClientConfig struct {
	ManagerIP string
	SocketURL string
	Sharded   bool
	Cache     RedisConfig
}

type RedisConfig struct {
}
