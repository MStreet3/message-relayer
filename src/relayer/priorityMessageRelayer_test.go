package relayer

import (
	"sync"
	"testing"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/utils"
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

	msgsToBroadcast := mr.Dequeue()

	expectedMessageTypesInOrder := []domain.MessageType{
		domain.StartNewRound,
		domain.StartNewRound,
		domain.ReceivedAnswer,
	}

	/* assertions */
	// each message type should be of the right type
	i := 0
	consumeAndAssert := func(results <-chan domain.Message) {
		for result := range results {
			utils.DPrintf("Received: %d\n", result)
			require.Equal(t, expectedMessageTypesInOrder[i], result.Type)
			i++
		}
		utils.DPrintf("Done receiving!")
	}

	consumeAndAssert(msgsToBroadcast)

}
