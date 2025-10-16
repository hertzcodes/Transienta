package client

type Client struct {
	config ClientConfig
}

func New(config ClientConfig) *Client {
	return &Client{
		config: config,
	}
}
