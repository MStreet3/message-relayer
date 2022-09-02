package mailbox

import (
	"context"
	"log"
	"sync"

	"github.com/mstreet3/message-relayer/domain"
)

type MessageMailbox struct {
	mu          sync.RWMutex
	cap         int
	stack       Stack[domain.Message]
	emptier     Emptier[domain.Message]
	timestamper TimeStamper
	emptiedAt   int64
}

func NewMessageMailbox(
	c int,
	empt Emptier[domain.Message],
	stack Stack[domain.Message],
) *MessageMailbox {
	return &MessageMailbox{
		mu:      sync.RWMutex{},
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
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.emptiedAt
}

func (q *MessageMailbox) empty() []domain.Message {
	msgs := q.emptier.Empty()
	q.setEmptiedAt()
	return msgs
}

func (q *MessageMailbox) setEmptiedAt() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.emptiedAt = q.getTimestamper().Timestamp()
}

func (q *MessageMailbox) Postmark(msg *domain.Message) {
	q.mu.Lock()
	defer q.mu.Unlock()
	msg.Timestamp = q.getTimestamper().Timestamp()
}

func (q *MessageMailbox) getTimestamper() TimeStamper {
	if q.timestamper == nil {
		q.timestamper = NewTimeStamper()
	}
	return q.timestamper
}
