package lifoqueue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_pop_empty_queue_returns_nil(t *testing.T) {
	var expected *interface{}
	queue := NewLIFOQueue[interface{}]()
	latest, present := queue.Pop()
	require.Equal(t, expected, latest)
	require.Equal(t, false, present)
}
func Test_queue_is_LIFO(t *testing.T) {
	queue := NewLIFOQueue[[]byte]()
	messages := [][]byte{
		[]byte("first in"),
		[]byte("last in"),
	}
	for _, msg := range messages {
		queue.Push(msg)
	}
	firstOut, _ := queue.Pop()
	lastOut, _ := queue.Pop()
	require.Equal(t, []byte("last in"), *firstOut)
	require.Equal(t, []byte("first in"), *lastOut)
}
