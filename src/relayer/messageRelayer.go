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
	ReadAndRelay()
}

type MakeMessageRelayerServer = func(network.NetworkSocket) MessageRelayerServer

type MessageEnquer interface {
	Enqueue(msg domain.Message) error
	Len(msgType domain.MessageType) int
}

type PriorityMessageRelayerServer interface {
	MessageEnquer
	MessageRelayerServer
	DequeueAndRelay()
}
