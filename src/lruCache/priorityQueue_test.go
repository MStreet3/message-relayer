package lruCache

import (
	"testing"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/stretchr/testify/require"
)

func Test_pop_empty_queue_returns_nil(t *testing.T) {
	var expected *domain.Message
	queue := MessagePriorityQueue{Capacity: domain.PriorityQueueCapacity}
	latest, present := queue.Pop()
	require.Equal(t, expected, latest)
	require.Equal(t, false, present)
}

func Test_queue_is_LIFO(t *testing.T) {
	queue := MessagePriorityQueue{Capacity: domain.PriorityQueueCapacity}
	messages := []domain.Message{
		{Type: domain.StartNewRound,
			Data: []byte("first in")},
		{Type: domain.StartNewRound,
			Data: []byte("last in")},
	}
	for _, msg := range messages {
		queue.Push(msg)
	}
	firstOut, _ := queue.Pop()
	lastOut, _ := queue.Pop()
	require.Equal(t, []byte("last in"), firstOut.Data)
	require.Equal(t, []byte("first in"), lastOut.Data)
}
