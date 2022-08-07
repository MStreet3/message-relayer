package relayer

import (
	"github.com/mstreet3/message-relayer/domain"
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
