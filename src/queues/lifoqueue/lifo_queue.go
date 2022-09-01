package lifoqueue

import (
	"container/list"
	"sync"
)

type LIFOQueue[T any] struct {
	mu    sync.Mutex
	queue *list.List
}

func NewLIFOQueue[T any]() *LIFOQueue[T] {
	return &LIFOQueue[T]{
		mu:    sync.Mutex{},
		queue: list.New(),
	}
}

func (q *LIFOQueue[T]) Len() int {
	return q.queue.Len()
}

func (q *LIFOQueue[T]) PushFront(msg T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.queue.PushFront(&msg)
}

func (q *LIFOQueue[T]) Pop() (*T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.pop()
}

func (q *LIFOQueue[T]) pop() (*T, bool) {
	if e := q.queue.Front(); e != nil {
		q.queue.Remove(e)
		return e.Value.(*T), true
	}

	return nil, false
}

func (q *LIFOQueue[T]) Empty() []T {
	q.mu.Lock()
	defer q.mu.Unlock()

	vals := make([]T, 0)
	for val, ok := q.pop(); ok; val, ok = q.pop() {
		vals = append(vals, *val)
	}

	return vals
}
