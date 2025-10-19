package client

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/hertzcodes/transienta/go-sdk/internal/comms"
	"github.com/hertzcodes/transienta/go-sdk/internal/utils"
)

var (
	instance *Client
	once     sync.Once
)

type Client struct {
	deps   map[uuid.UUID][]string
	mu     *sync.RWMutex
	config ClientConfig
}

func Instance(config ...ClientConfig) *Client {
	once.Do(
		func() {
			if len(config) == 0 {
				panic("TRANSIENTA: you must provide configuration on initialization")
			}
			instance = &Client{
				deps:   make(map[uuid.UUID][]string),
				config: config[0],
			}
		})
	return instance
}

func (c *Client) StartRequest(req []byte, caller string, base context.Context) *Ctx {
	ctx := newCtx(caller, base)
	ctx.args = utils.Hash(req)

	// send startRequest with args

	// maybe there's no need to lock or unlock? cause ids are unique
	c.mu.Lock()
	c.deps[ctx.id] = make([]string, 0)
	c.mu.Unlock()

	return ctx
}

func (c *Client) EndRequest(ctx *Ctx, resp any) {
	deps := c.deps[ctx.id]
	delete(c.deps, ctx.id)

	req := comms.Request{
		Type:   comms.EndRequest,
		Args:   ctx.args,
		Caller: ctx.caller,
		Deps:   deps,
	}

	// send the end request
	_ = req
}
