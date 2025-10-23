package client

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/hertzcodes/transienta/go-sdk/internal/comms"
	"github.com/hertzcodes/transienta/go-sdk/internal/utils"
	"go.nanomsg.org/mangos/v3"
)

var (
	instance *Client
	once     sync.Once
)

type Client struct {
	deps   map[uuid.UUID][]string
	mu     sync.RWMutex
	config ClientConfig
	socket mangos.Socket
}

func New(config ClientConfig) *Client {
	once.Do(func() {
		instance = &Client{
			deps:   make(map[uuid.UUID][]string),
			config: config,
		}
		sock, err := comms.Connect(config.SocketURL)
		if err != nil {
			return
		}

		instance.socket = sock
	})

	return instance
}

func GetInstance() *Client {
	if instance == nil {
		panic("TRANSIENTA: Initialize an instance with New() before calling this function")
	}
	return instance
}

func (c *Client) Start(req []byte, caller string, base context.Context) context.Context {
	ctx := newCtx(caller, base)
	ctx.args = utils.Hash([]byte(c.config.ManagerIP), req)

	c.mu.Lock()
	c.deps[ctx.id] = make([]string, 0)
	c.mu.Unlock()

	request := &comms.StartRequest{
		Number: comms.Start,
		Args:   ctx.args,
	}

	if c.socket != nil {
		c.socket.Send(request.Serialize())
	}
	return ctx
}

// call this when you are reading from a database key or calling another service
func (c *Client) AddDependency(ct context.Context, key string) error {
	ctx, ok := ct.(*Ctx)
	if !ok {
		panic("invalid type of context provided. make sure it's originated from a client.Ctx")
	}
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

	request := comms.InvalidationRequest{
		Number: comms.Invalidation,
		Key:    key,
	}
	if c.socket != nil {
		c.socket.Send(request.Serialize())
	}
}

func (c *Client) Finish(ct context.Context, resp any) {
	ctx, ok := ct.(*Ctx)
	if !ok {
		panic("invalid type of context provided. make sure it's originated from a client.Ctx")
	}
	c.mu.Lock()
	deps := c.deps[ctx.id]
	delete(c.deps, ctx.id)
	c.mu.Unlock()

	// if it's valid then send the end request
	request := comms.EndRequest{
		Number: comms.End,
		Args:   ctx.args,
		Time:   ctx.timestamp,
		Caller: ctx.caller,
		Deps:   deps,
	}

	// send the end request
	if c.socket != nil {
		c.socket.Send(request.Serialize())
	}
}
