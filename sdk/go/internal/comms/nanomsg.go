package comms

import (
	"fmt"
	"time"

	nng "go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pair"
	_ "go.nanomsg.org/mangos/v3/transport/all"
)

func SendRequest(socket nng.Socket, request Request) {
	counter := 0
	for err := socket.Send(request.Serialize()); err != nil; counter++ {
		if counter > 3 {
			// log the error
			break
		}
	}
}

func Connect(url string) (nng.Socket, error) {
	var sock nng.Socket
	var err error
	if sock, err = pair.NewSocket(); err != nil {
		return nil, fmt.Errorf("couldn't create socket. %s", err)
	}
	sock.SetOption(nng.OptionSendDeadline, 1*time.Second) // TODO: find a formula for this based on queue size
	sock.SetOption(nng.OptionRecvDeadline, 1*time.Second)
	if err = sock.Dial(url); err != nil {
		return nil, fmt.Errorf("couldn't dial on URL: %s, MESSAGE: %s", url, err)
	}

	return sock, nil
}
