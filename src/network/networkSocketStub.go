package network

import (
	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/errs"
)

type NetworkResponse struct {
	Message domain.Message
	Error   error
}

type NetworkSocketStub struct {
	Cursor    int
	Responses []NetworkResponse
}

func (n *NetworkSocketStub) Read() (domain.Message, error) {
	if n.Cursor < len(n.Responses) {
		response := n.Responses[n.Cursor]
		n.Cursor++
		return response.Message, response.Error
	}
	return domain.Message{}, errs.FatalSocketError{}
}

func (n *NetworkSocketStub) ResetCursor() {
	n.Cursor = 0
}

func NewNetworkSocketStub(responses []NetworkResponse) *NetworkSocketStub {
	return &NetworkSocketStub{
		Cursor:    0,
		Responses: responses,
	}
}
