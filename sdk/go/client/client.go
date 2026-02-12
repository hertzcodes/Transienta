package client

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/hertzcodes/easy-outbox/go/outbox"
	"github.com/hertzcodes/transienta/go-sdk/internal/comms"
	"github.com/hertzcodes/transienta/go-sdk/internal/utils"
	"go.nanomsg.org/mangos/v3"
)

var (
	instance *Client
	once     sync.Once
)

type Client struct {
	deps              map[uuid.UUID][]string
	mu                sync.RWMutex
	config            ClientConfig
	socket            mangos.Socket
	outbox            *outbox.OutBox
}

func New(config ClientConfig) *Client {
	if config.On {
		once.Do(func() {
			instance = &Client{
				deps:   make(map[uuid.UUID][]string),
				config: config,
			}

			// set socket (PAIR)
			sock, err := comms.Connect(config.SocketURL)
			if err != nil {
				return
			}
			instance.socket = sock

			// set outbox
			o, err := outbox.New(outbox.DBTypePebble, "./.transienta_db")
			if err != nil {
				panic(fmt.Sprintf("TRANSIENTA: failed to create outbox. is database configuration correct?\n type: %s , path: %s, err: %s", "pebble", "./transienta_db", err.Error()))
			}
			instance.outbox = o
			ch := make(chan string, 20000)
			instance.outbox.WithStream(ch)
			instance.sendInvalidationsFromOutbox(ch)
		})
	} else {
		instance = &Client{
			config: config,
		}
	}
	return instance
}

func GetInstance() *Client {
	if instance == nil {
		panic("TRANSIENTA: Initialize an instance with New() before calling this function")
	}
	return instance
}

func (c *Client) Start(req []byte, caller string, base context.Context) context.Context {
	if c.config.On {
		ctx := newCtx(caller, base)
		ctx.args = utils.Hash([]byte(c.config.ManagerIP), req)

		request := &comms.StartRequest{
			ID: ctx.id.String(),
		}

		if c.socket != nil {
			// TODO: should it be blocking?
			c.socket.Send(request.Serialize())
		}
		c.mu.Lock()
		c.deps[ctx.id] = make([]string, 0)
		c.mu.Unlock()
		return ctx
	}
	return base
}

// call this when you are reading from a database key or calling another service
func (c *Client) AddDependency(ct context.Context, key string) error {
	if c.config.On {
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
	return nil
}

// call when you are updating or writing to a key. this invalidates the key from cache
func (c *Client) Invalidate(key string) {
	if c.config.On {
		// COULD THIS BE REMOVED?
		request := comms.InvalidationRequest{
			Key: key,
		}
		if c.socket != nil {
			if err := c.socket.Send(request.Serialize()); err != nil {
				// sends message to outbox if request fails
				if err := c.outbox.SetMessage(key, nil); err != nil {
					// log
				}
			}
		}
	}
}

func (c *Client) sendInvalidationsFromOutbox(ch chan string) {
	go func() {
		for key := range ch {
			request := comms.InvalidationRequest{
				Key: key,
			}
			if c.socket != nil {
				if err := c.socket.Send(request.Serialize()); err == nil {
					c.outbox.Delete(key)
				}
			}
		}
	}()
}

func (c *Client) Finish(ct context.Context, resp any) {
	if c.config.On {
		ctx, ok := ct.(*Ctx)
		if !ok {
			panic("invalid type of context provided. make sure it's originated from a client.Ctx")
		}
		c.mu.Lock()
		deps := c.deps[ctx.id]
		delete(c.deps, ctx.id)
		c.mu.Unlock()
		go func() {
			// if it's valid then send the end request
			request := comms.EndRequest{
				ID:     ctx.id.String(),
				Args:   ctx.args,
				Caller: ctx.caller,
				Deps:   deps,
			}

			// send the end request
			if c.socket != nil {
				c.socket.Send(request.Serialize())
			}
		}()
	}
}
