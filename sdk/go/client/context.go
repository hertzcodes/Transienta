package client

import (
	"context"

	"github.com/google/uuid"
)

type Ctx struct {
	id        uuid.UUID
	caller    string
	args      uint32
	context.Context
}

func newCtx(caller string, base context.Context) *Ctx {
	return &Ctx{
		id:      uuid.New(),
		caller:  caller,
		Context: base,
	}
}
