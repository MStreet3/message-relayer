package relayer

import (
	"context"

	"github.com/mstreet3/message-relayer/domain"
)

type Subscriber interface {
	Subscribe(context.Context, domain.MessageType) <-chan domain.Message
}
type MessageRelayer interface {
	Subscriber
	Errors() <-chan error
}

type MessageRelayerServer interface {
	MessageRelayer
	Start(context.Context) <-chan struct{} // serve
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
