package network

import (
	"time"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/errs"
)

type NetworkResponse struct {
	Message *domain.Message
	Error   error
}

type NetworkSocketStub struct {
	Cursor    int
	Responses []NetworkResponse
}

func (n *NetworkSocketStub) Read() (*domain.Message, error) {
	if n.Cursor < len(n.Responses) {
		<-time.After(30 * time.Millisecond)
		response := n.Responses[n.Cursor]
		response.Message.Timestamp = time.Now().UTC().UnixNano()
		n.Cursor++
		return response.Message, response.Error
	}
	return nil, errs.FatalSocketError{}
}

func (n *NetworkSocketStub) Restart() error {
	n.Cursor = 0
	return nil
}

func NewNetworkSocketStub(responses []NetworkResponse) RestartNetworkReader {
	return &NetworkSocketStub{
		Cursor:    0,
		Responses: responses,
	}
}
