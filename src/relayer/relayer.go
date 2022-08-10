package relayer

import (
	"context"

	"github.com/mstreet3/message-relayer/domain"
)

type Subscriber[T interface{}, U interface{}] interface {
	Subscribe(T) (<-chan U, func())
}
type Enqueuer[T interface{}] interface {
	Enqueue(T)
}
type Broadcaster[T interface{}] interface {
	Broadcast(ch <-chan T)
}
type Dequeuer[T interface{}] interface {
	Dequeue() <-chan T
}

type MessageRelayer interface {
	Subscriber[domain.MessageType, domain.Message]
	Errors() <-chan error
	Start(context.Context) <-chan struct{}
}
type PriorityMessageRelayer interface {
	Enqueuer[domain.Message]
	Dequeuer[domain.Message]
	Broadcaster[domain.Message]
	MessageRelayer
}
