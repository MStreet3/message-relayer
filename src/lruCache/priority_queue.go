package lruCache

import (
	"container/list"

	"github.com/mstreet3/message-relayer/domain"
)

type PriorityQueue interface {
	Pop() (*domain.Message, bool)
	Push(domain.Message)
}

type MessagePriorityQueue struct {
	Capacity int
	Queue    *list.List
}

func NewMessagePriorityQueue(c int) PriorityQueue {
	return &MessagePriorityQueue{
		Capacity: c,
		Queue:    list.New(),
	}
}

func (q *MessagePriorityQueue) Pop() (*domain.Message, bool) {
	if e := q.Queue.Front(); e != nil {
		q.Queue.Remove(e)
		return e.Value.(*domain.Message), true
	}

	return nil, false
}

func (q *MessagePriorityQueue) Push(msg domain.Message) {
	if q.Queue.Len() == q.Capacity {
		e := q.Queue.Back()
		q.Queue.Remove(e)
	}
	q.Queue.PushFront(&msg)
}
