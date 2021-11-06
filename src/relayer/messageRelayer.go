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
	ListenAndRelay()
}

type MakeMessageRelayerServer = func(network.NetworkSocket) MessageRelayerServer
