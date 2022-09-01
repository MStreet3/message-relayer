package relayer

import (
	"errors"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
)

var StartNewRoundResponse = network.NetworkResponse{Message: &domain.Message{
	Type: domain.StartNewRound,
	Data: nil,
},
	Error: nil,
}

var ReceivedAnswerResponse = network.NetworkResponse{Message: &domain.Message{
	Type: domain.ReceivedAnswer,
	Data: nil,
},
	Error: nil,
}

var NetworkErrorResponse = network.NetworkResponse{
	Error: errors.New("network unavailable"),
}

type testMailbox struct {
	msgs []domain.Message
}
