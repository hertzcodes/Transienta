package comms

import (
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/hertzcodes/transienta/go-sdk/internal/fbs"
)

type RequestType uint8

const (
	Start RequestType = iota
	Invalidation
	End
)

type Request interface {
	Serialize() []byte
}

type StartRequest struct {
	Number RequestType
	Args   uint32
}

func (s *StartRequest) Serialize() []byte {
	builder := flatbuffers.NewBuilder(0)
	fbs.StartRequestStart(builder)
	fbs.StartRequestAddNumber(builder, fbs.RequestTypeStart)
	fbs.StartRequestAddArgs(builder, s.Args)

	offset := fbs.StartRequestEnd(builder)
	builder.Finish(offset)
	return builder.FinishedBytes()
}

type EndRequest struct {
	Number RequestType
	Args   uint32
	Caller string
	Deps   []string
	Resp   []byte
}

func (e *EndRequest) Serialize() []byte {
	builder := flatbuffers.NewBuilder(0)

	callerOffset := builder.CreateString(e.Caller)
	depsOffsets := make([]flatbuffers.UOffsetT, len(e.Deps))
	for i, dep := range e.Deps {
		depsOffsets[i] = builder.CreateString(dep)
	}
	fbs.EndRequestStartDepsVector(builder, len(e.Deps))
	for i := len(e.Deps) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(depsOffsets[i])
	}
	depsVector := builder.EndVector(len(e.Deps))
	respVector := builder.CreateByteVector(e.Resp)

	fbs.EndRequestStart(builder)
	fbs.EndRequestAddNumber(builder, fbs.RequestTypeEnd)
	fbs.EndRequestAddArgs(builder, e.Args)
	fbs.EndRequestAddCaller(builder, callerOffset)
	fbs.EndRequestAddDeps(builder, depsVector)
	fbs.EndRequestAddResp(builder, respVector)

	offset := fbs.EndRequestEnd(builder)
	builder.Finish(offset)
	return builder.FinishedBytes()
}

type InvalidationRequest struct {
	Number RequestType
	Key    string
}

func (i *InvalidationRequest) Serialize() []byte {
	builder := flatbuffers.NewBuilder(0)
	keyOffset := builder.CreateString(i.Key)

	fbs.InvalidationRequestStart(builder)
	fbs.InvalidationRequestAddNumber(builder, fbs.RequestTypeInvalidation)
	fbs.InvalidationRequestAddKey(builder, keyOffset)

	offset := fbs.InvalidationRequestEnd(builder)
	builder.Finish(offset)
	return builder.FinishedBytes()
}
