package relayer

import (
	"errors"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
)

type MessageRelayer interface {
	SubscribeToMessage(msgType domain.MessageType, ch chan<- domain.Message)
}

type MessageRelayerServer interface {
	MessageRelayer
	ReadAndRelay() // serve
}
type MessageEnquer interface {
	Enqueue(msg domain.Message)
	Len(msgType domain.MessageType) int
}
type MessageBroadcaster interface {
	Broadcast(ch <-chan domain.Message)
}
type MessageDequer interface {
	Dequeue() <-chan domain.Message
	MessageBroadcaster
}
type PriorityMessageRelayerServer interface {
	MessageEnquer
	MessageDequer
	MessageRelayerServer
}

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
type MakeMessageRelayerServer = func(network.NetworkSocket) MessageRelayerServer
type MakePriorityMessageRelayerServer = func(network.NetworkSocket) PriorityMessageRelayerServer
type MessageRelayerServerTestCase struct {
	Name  string
	Maker interface{}
}
