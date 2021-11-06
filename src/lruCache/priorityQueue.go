package lruCache

import "github.com/mstreet3/message-relayer/domain"

type MessagePriorityQueue struct {
	Capacity int
	Queue    DblLinkedList
}

func (q *MessagePriorityQueue) Pop() domain.Message {
	node := q.Queue.Head
	q.Queue.DeleteListHead()
	return node.Value.(domain.Message)
}

func (q *MessagePriorityQueue) Push(msg domain.Message) {
	n := &Node{
		Value: msg,
	}
	if q.Queue.Length == q.Capacity {
		q.Queue.DeleteListTail()
	}
	q.Queue.SetListHead(n)
}
