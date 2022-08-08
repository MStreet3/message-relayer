package relayer

import (
	"context"

	"github.com/google/uuid"
	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/utils"
)

type MessageObserver interface {
	Observe(context.Context, domain.Message) error
}
type SubscriberAddress struct {
	msgCh chan<- domain.Message
	ID    uuid.UUID
}

func NewSubscriberAddress(ch chan<- domain.Message) MessageObserver {
	return &SubscriberAddress{
		msgCh: ch,
		ID:    uuid.New(),
	}
}

func (sa *SubscriberAddress) Observe(ctx context.Context, msg domain.Message) error {
	select {
	case sa.msgCh <- msg:
		utils.DPrintf("%s received message of type %s\n", sa.ID.String(), msg.Type)
	case <-ctx.Done():
	default:
		utils.DPrintf("%s is busy, dropping message of type %s\n", sa.ID.String(), msg.Type)
	}
	return nil
}
