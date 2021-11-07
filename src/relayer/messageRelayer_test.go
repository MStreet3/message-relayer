package relayer

import (
	"errors"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
)

var StartNewRoundResponse = network.NetworkResponse{Message: domain.Message{
	Type: domain.StartNewRound,
	Data: nil,
},
	Error: nil,
}

var ReceivedAnswerResponse = network.NetworkResponse{Message: domain.Message{
	Type: domain.ReceivedAnswer,
	Data: nil,
},
	Error: nil,
}

var NetworkErrorResponse = network.NetworkResponse{Message: domain.Message{},
	Error: errors.New("network unavailable"),
}

/*
	todo: fix the dependecy injection of the makeTestCase function
*/
type MessageRelayerServerTestCase struct {
	Name  string
	Maker interface{}
}
