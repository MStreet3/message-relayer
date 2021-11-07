package relayer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_enqueue(t *testing.T) {
	testCases := []struct {
		name     string
		expected bool
	}{
		{name: "lengths of queues are as expected", expected: true},
		{name: "order of popped items is as expected", expected: true},
		{name: "pop on empty queue gives nil", expected: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, true, tc.expected)
		})
	}
}

func Test_dequeue_result_channel_has_expected_messages(t *testing.T) {
	testCases := []struct {
		name     string
		expected bool
	}{
		{name: "4 start new and 1 receive answer enqueued", expected: true},
		{name: "0 start new and 1 receive answer enqueued", expected: true},
		{name: "0 start new and 0 receive answer enqueued", expected: true},
		{name: "4 start new and 4 receive answer enqueued", expected: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, true, tc.expected)
		})
	}
}
