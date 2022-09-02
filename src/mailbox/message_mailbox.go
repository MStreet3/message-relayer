package mailbox

import (
	"context"
	"log"
	"sync"

	"github.com/mstreet3/message-relayer/domain"
)

type MessageMailbox struct {
	mu          sync.Mutex
	cap         int
	stack       Stack[domain.Message]
	emptier     Emptier[domain.Message]
	timestamper TimeStamper
}

func NewMessageMailbox(
	c int,
	empt Emptier[domain.Message],
	stack Stack[domain.Message],
) *MessageMailbox {
	return &MessageMailbox{
		mu:      sync.Mutex{},
		cap:     c,
		emptier: empt,
		stack:   stack,
	}
}

func (q *MessageMailbox) Add(msg domain.Message) {
	if msg.Timestamp < q.EmptiedAt() {
		return
	}

	if q.stack.Len() == q.cap {
		found, ok := q.stack.PopBack()
		if !ok {
			log.Fatal("expected an element at the bottom of the stack")
		}
		if found.Timestamp > msg.Timestamp {
			q.stack.PushFront(*found)
		}
		return
	}

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

func (q *MessageMailbox) EmptiedAt() int64 {
	return q.getTimestamper().GetTimestamp()
}

func (q *MessageMailbox) empty() []domain.Message {
	q.getTimestamper().SetTimestamp()
	return q.emptier.Empty()
}

func (q *MessageMailbox) getTimestamper() TimeStamper {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.timestamper == nil {
		q.timestamper = NewTimeStamper()
	}
	return q.timestamper
}
