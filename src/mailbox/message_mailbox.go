package mailbox

import (
	"context"

	"github.com/mstreet3/message-relayer/domain"
)

type MessageMailbox struct {
	cap     int
	emptier Emptier[domain.Message]
	stack   Stack[domain.Message]
}

func NewMessageMailbox(c int, empt StackEmptier[domain.Message]) *MessageMailbox {
	return &MessageMailbox{
		cap:     c,
		emptier: empt,
		stack:   empt,
	}
}

func (q *MessageMailbox) Add(msg domain.Message) {
	q.stack.PushFront(msg)
}

// Empty drains the queue and puts all found values onto a channel
func (q *MessageMailbox) Empty(ctx context.Context) <-chan domain.Message {
	msgCh := make(chan domain.Message, 1)

	go func() {
		defer close(msgCh)
		for _, msg := range q.empty() {
			select {
			case <-ctx.Done():
				return
			case msgCh <- msg:
			default:
				// drop messages on the floor if no listener
			}
		}
	}()

	return msgCh
}

func (q *MessageMailbox) empty() []domain.Message {
	return q.emptier.Empty()
}
