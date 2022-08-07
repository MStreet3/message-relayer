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
	Errors() <-chan error
}

type MessageRelayerServer interface {
	MessageRelayer
	Start() // serve
	Stop()
}
type MessageEnquer interface {
	Enqueue(msg domain.Message)
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
type MakeMessageRelayerServer = func(network.NetworkReader) MessageRelayerServer
type MakePriorityMessageRelayerServer = func(network.NetworkReader) PriorityMessageRelayerServer
type MessageRelayerServerTestCase struct {
	Name  string
	Maker interface{}
}
