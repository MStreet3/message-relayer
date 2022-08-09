package relayer

import (
	"github.com/google/uuid"
	"github.com/mstreet3/message-relayer/domain"
)

type MessageObserver interface {
	Observe(domain.Message) error
}
type SubscriberAddress struct {
	ID uuid.UUID
	f  func(domain.Message)
}

func NewSubscriberAddress(id uuid.UUID, f func(domain.Message)) MessageObserver {
	return &SubscriberAddress{
		f:  f,
		ID: id,
	}
}

func (sa *SubscriberAddress) Observe(msg domain.Message) error {
	sa.f(msg)
	return nil
}
