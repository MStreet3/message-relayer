package relayer

import (
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

type MakeMessageRelayerServer = func(network.NetworkSocket) MessageRelayerServer
type MakePriorityMessageRelayerServer = func(network.NetworkSocket) PriorityMessageRelayerServer

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
