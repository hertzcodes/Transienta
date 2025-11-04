package comms

import (
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/hertzcodes/transienta/go-sdk/internal/fbs"
)

type RequestType uint8
type Request interface {
	Serialize() []byte
}

type StartRequest struct {
	ID string
}

func (s *StartRequest) Serialize() []byte {
	builder := flatbuffers.NewBuilder(0)
	idOffset := builder.CreateString(s.ID)
	fbs.StartRequestStart(builder)
	fbs.StartRequestAddId(builder, idOffset)

	offset := fbs.StartRequestEnd(builder)
	fbs.RequestStart(builder)
	fbs.RequestAddRequestType(builder,  fbs.RequestUnionStartRequest)
	fbs.RequestAddRequest(builder, offset)
	req := fbs.RequestEnd(builder)
	builder.Finish(req)
	return builder.FinishedBytes()
}

type EndRequest struct {
	ID     string
	Args   uint32
	Caller string
	Deps   []string
	Resp   []byte
}

func (e *EndRequest) Serialize() []byte {
	builder := flatbuffers.NewBuilder(0)
	idOffset := builder.CreateString(e.ID)
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
	fbs.EndRequestAddArgs(builder, e.Args)
	fbs.EndRequestAddId(builder, idOffset)
	fbs.EndRequestAddCaller(builder, callerOffset)
	fbs.EndRequestAddDeps(builder, depsVector)
	fbs.EndRequestAddResp(builder, respVector)

	offset := fbs.EndRequestEnd(builder)
	fbs.RequestStart(builder)
	fbs.RequestAddRequestType(builder,  fbs.RequestUnionEndRequest)
	fbs.RequestAddRequest(builder, offset)
	req := fbs.RequestEnd(builder)
	builder.Finish(req)
	return builder.FinishedBytes()
}

type InvalidationRequest struct {
	Key string
}

func (i *InvalidationRequest) Serialize() []byte {
	builder := flatbuffers.NewBuilder(0)
	keyOffset := builder.CreateString(i.Key)

	fbs.InvalidationRequestStart(builder)
	fbs.InvalidationRequestAddKey(builder, keyOffset)

	offset := fbs.InvalidationRequestEnd(builder)
	fbs.RequestStart(builder)
	fbs.RequestAddRequestType(builder,  fbs.RequestUnionInvalidationRequest)
	fbs.RequestAddRequest(builder, offset)
	req := fbs.RequestEnd(builder)
	builder.Finish(req)
	return builder.FinishedBytes()
}
