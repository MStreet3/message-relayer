package relayer

import (
	"github.com/mstreet3/message-relayer/domain"
)

type MessageRelayer interface {
	SubscribeToMessage(msgType domain.MessageType, ch chan<- domain.Message)
}

type MessageRelayerServer interface {
	MessageRelayer
	ListenAndRelay()
}
