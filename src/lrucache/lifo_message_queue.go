package lrucache

import (
	"container/list"
	"context"
	"sync"

	"github.com/mstreet3/message-relayer/domain"
)

type LIFOMessageQueue struct {
	mu    sync.Mutex
	cap   int
	queue *list.List
}

func NewMessagePriorityQueue(c int) *LIFOMessageQueue {
	return &LIFOMessageQueue{
		cap:   c,
		queue: list.New(),
	}
}

func (q *LIFOMessageQueue) Len() int {
	return q.queue.Len()
}

func (q *LIFOMessageQueue) Pop() (*domain.Message, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.pop()
}

func (q *LIFOMessageQueue) pop() (*domain.Message, bool) {
	if e := q.queue.Front(); e != nil {
		q.queue.Remove(e)
		return e.Value.(*domain.Message), true
	}

	return nil, false
}

func (q *LIFOMessageQueue) Push(msg domain.Message) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.queue.Len() == q.cap {
		e := q.queue.Back()
		q.queue.Remove(e)
	}
	q.queue.PushFront(&msg)
}

func (q *LIFOMessageQueue) Add(msg domain.Message) {
	q.Push(msg)
}

func (q *LIFOMessageQueue) Empty(ctx context.Context) <-chan domain.Message {
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

func (q *LIFOMessageQueue) empty() []domain.Message {
	q.mu.Lock()
	defer q.mu.Unlock()

	msgs := make([]domain.Message, 0)
	for msg, ok := q.pop(); ok; msg, ok = q.pop() {
		msgs = append(msgs, *msg)
	}

	return msgs
}
