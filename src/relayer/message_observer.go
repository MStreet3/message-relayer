package relayer

import (
	"github.com/google/uuid"
	"github.com/mstreet3/message-relayer/domain"
)

type MessageObserver Observer[domain.Message]

type MessageHandler func(domain.Message) error

type messageObserver struct {
	ID     uuid.UUID
	handle MessageHandler
}

func NewMessageObserver(id uuid.UUID, f MessageHandler) MessageObserver {
	return &messageObserver{
		handle: f,
		ID:     id,
	}
}

func (mo *messageObserver) Observe(msg domain.Message) error {
	return mo.handle(msg)
}
