package relayer

import (
	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
)

type Subscriber interface {
	Subscribe(domain.MessageType) <-chan domain.Message
}
type MessageRelayer interface {
	Subscriber
}

type MessageRelayerServer interface {
	MessageRelayer
	Start() // serve
	Stop()
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

/*
	todo: fix the dependecy injection of the makeTestCase function
*/
type MakeMessageRelayerServer = func(network.NetworkSocket) MessageRelayerServer
type MakePriorityMessageRelayerServer = func(network.NetworkSocket) PriorityMessageRelayerServer
type MessageRelayerServerTestCase struct {
	Name  string
	Maker interface{}
}
