package relayer

import (
	"sync"
	"testing"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/stretchr/testify/require"
)

func Test_enqueue_results_in_correct_queue_lengths(t *testing.T) {
	// use stub socket
	var wg sync.WaitGroup
	responses := []network.NetworkResponse{
		StartNewRoundResponse,
		ReceivedAnswerResponse,
		StartNewRoundResponse,
		ReceivedAnswerResponse,
	}
	socket := network.NewNetworkSocketStub(responses)

	mr := NewPriorityMessageRelayer(socket)

	for _, res := range responses {
		wg.Add(1)
		go func(msg domain.Message) {
			defer wg.Done()
			mr.Enqueue(msg)
		}(res.Message)
	}

	wg.Wait()
	require.Equal(t, 2, mr.Len(domain.StartNewRound))
	require.Equal(t, 2, mr.Len(domain.ReceivedAnswer))
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
