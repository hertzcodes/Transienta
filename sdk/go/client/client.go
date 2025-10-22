package client

import (
	"context"
	"errors"
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
	mu     sync.RWMutex
	config ClientConfig
}

func New(config ClientConfig) *Client {
	once.Do(func() {
		instance = &Client{
			deps:   make(map[uuid.UUID][]string),
			config: config,
		}
	})

	return instance
}

func GetInstance() *Client {
	if instance == nil {
		panic("TRANSIENTA: Initialize an instance with New() before calling this function")
	}
	return instance
}

func (c *Client) StartRequest(req []byte, caller string, base context.Context) *Ctx {
	ctx := newCtx(caller, base)
	ctx.args = utils.Hash(req)

	request := comms.Request{
		Type: comms.StartRequest,
		Args: ctx.args,
	}

	_ = request

	// maybe there's no need to lock or unlock? cause ids are unique
	c.mu.Lock()
	c.deps[ctx.id] = make([]string, 0)
	c.mu.Unlock()

	return ctx
}

// call this when you are reading from a database key or calling another service
func (c *Client) AddDependency(ctx *Ctx, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.deps[ctx.id]; exists {
		c.deps[ctx.id] = append(c.deps[ctx.id], key)
		return nil
	}
	return errors.New("invalid ctx (id not found in requests)")
}

// call when you are updating or writing to a key. this invalidates the key from cache
func (c *Client) Invalidate(key string) {

	request := comms.Request{
		Type: comms.InvalidationRequest,
		Key:  key,
	}

	_ = request
}

func (c *Client) EndRequest(ctx *Ctx, resp any) {
	c.mu.Lock()
	deps := c.deps[ctx.id]
	delete(c.deps, ctx.id)
	c.mu.Unlock()

	request := comms.Request{
		Type:   comms.EndRequest,
		Args:   ctx.args,
		Caller: ctx.caller,
		Deps:   deps,
	}

	// send the end request
	_ = request
}
