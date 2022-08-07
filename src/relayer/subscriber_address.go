package relayer

import (
	"github.com/google/uuid"
	"github.com/mstreet3/message-relayer/domain"
)

type SubscriberAddress struct {
	msgCh chan<- domain.Message
	ID    uuid.UUID
}

func NewSubscriberAddress(ch chan<- domain.Message) *SubscriberAddress {
	return &SubscriberAddress{
		msgCh: ch,
		ID:    uuid.New(),
	}
}
