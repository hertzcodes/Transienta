package comms

type RequestType int

const (
	StartRequest RequestType = iota
	InvalidationRequest
	EndRequest
)

// this has to be generic later
type Request struct {
	Type RequestType
	Args uint32
	// for invalidations
	Key string
	// for end request only
	Deps   []string
	Resp   []byte
	Caller string
}
