package relayer

import (
	"context"

	"github.com/mstreet3/message-relayer/domain"
)

type Subscriber[T interface{}, U interface{}] interface {
	Subscribe(T) (<-chan U, func())
}
type MessageRelayer interface {
	Subscriber[domain.MessageType, domain.Message]
	Start(context.Context) <-chan struct{}
}
