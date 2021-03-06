package relayer

import (
	"fmt"
	"sync"
	"testing"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/utils"
	"github.com/stretchr/testify/require"
)

// each relayer should pass the shared test cases
var priorityMaker []MessageRelayerServerTestCase = []MessageRelayerServerTestCase{
	{
		Name:  "PriorityMessageRelayer:",
		Maker: NewPriorityMessageRelayer,
	},
}

func Test_enqueue_results_in_correct_queue_lengths_priority(t *testing.T) {
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

func Test_dequeue_result_channel_has_expected_messages_priority(t *testing.T) {
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

func Test_single_subscriber_priority(t *testing.T) {

	makeTestCaseName := func(n string) string {
		return fmt.Sprintf("%ssubscriber receives all messages of correct type", n)
	}

	makeTestCase := func(m MakePriorityMessageRelayerServer) func(t *testing.T) {
		return func(t *testing.T) {
			/* setup */
			var wg sync.WaitGroup
			// sub to start new round messages, bufferred because
			// busy channels get their messages dropped
			firstSubCh := make(chan domain.Message, 2)
			gotStartNewRound := 0

			// use stub socket
			responses := []network.NetworkResponse{
				StartNewRoundResponse,
				StartNewRoundResponse,
				ReceivedAnswerResponse,
			}
			socket := network.NewNetworkSocketStub(responses)

			mr := m(socket)

			// subscribe each channel to the relayer
			mr.SubscribeToMessage(domain.StartNewRound, firstSubCh)

			go mr.ReadAndRelay()

			/* actions */
			// read each message from the channel and increment the count
			wg.Add(1)
			go func() {
				for range firstSubCh {
					gotStartNewRound++
				}
				wg.Done()
			}()

			// assertions
			// count of received StartNewRound messages should be two
			wg.Wait()
			require.Equal(t, 2, gotStartNewRound)
		}
	}

	for _, tc := range priorityMaker {
		t.Run(makeTestCaseName(tc.Name), makeTestCase(tc.Maker.(MakePriorityMessageRelayerServer)))
	}

}
func Test_multiple_subscribers_priority(t *testing.T) {
	makeTestCaseName := func(n string) string {
		return fmt.Sprintf("%sall subscribers receive all messages of correct type", n)
	}

	makeTestCase := func(m MakePriorityMessageRelayerServer) func(t *testing.T) {
		return func(t *testing.T) {
			/* setup */
			var wg sync.WaitGroup
			// sub to start new round messages, bufferred because
			// busy channels get their messages dropped
			firstSubCh := make(chan domain.Message, 2)
			gotStartNewRound := 0

			secondSubCh := make(chan domain.Message, 2)
			gotReceivedAnswer := 0

			// use stub socket
			responses := []network.NetworkResponse{
				StartNewRoundResponse,
				ReceivedAnswerResponse,
				StartNewRoundResponse,
				ReceivedAnswerResponse,
			}
			socket := network.NewNetworkSocketStub(responses)

			mr := NewDefaultMessageRelayer(socket)

			// subscribe each channel to the relayer
			mr.SubscribeToMessage(domain.StartNewRound, firstSubCh)
			mr.SubscribeToMessage(domain.ReceivedAnswer, secondSubCh)

			go mr.ReadAndRelay()

			/* actions */
			// read each message from the channel and increment the count
			wg.Add(1)
			go func() {
				for range firstSubCh {
					gotStartNewRound++
				}
				wg.Done()
			}()

			wg.Add(1)
			go func() {
				for range secondSubCh {
					gotReceivedAnswer++
				}
				wg.Done()
			}()

			// assertions
			// count of received StartNewRound and ReceivedAnswer messages should be two
			wg.Wait()
			require.Equal(t, 2, gotStartNewRound)
			require.Equal(t, 2, gotReceivedAnswer)
		}
	}

	for _, tc := range priorityMaker {
		t.Run(makeTestCaseName(tc.Name), makeTestCase(tc.Maker.(MakePriorityMessageRelayerServer)))
	}

}

func Test_multiple_subscribers_and_errors_priority(t *testing.T) {
	makeTestCaseName := func(n string) string {
		return fmt.Sprintf("%srelayer continues with non fatal network errors", n)
	}

	makeTestCase := func(m MakePriorityMessageRelayerServer) func(t *testing.T) {
		return func(t *testing.T) {
			/* setup */
			var wg sync.WaitGroup
			// sub to start new round messages, bufferred because
			// busy channels get their messages dropped
			firstSubCh := make(chan domain.Message, 2)
			gotStartNewRound := 0

			secondSubCh := make(chan domain.Message, 2)
			gotReceivedAnswer := 0

			// use stub socket
			responses := []network.NetworkResponse{
				StartNewRoundResponse,
				ReceivedAnswerResponse,
				NetworkErrorResponse,
				NetworkErrorResponse,
				StartNewRoundResponse,
				ReceivedAnswerResponse,
				NetworkErrorResponse,
			}
			socket := network.NewNetworkSocketStub(responses)

			mr := NewDefaultMessageRelayer(socket)

			// subscribe each channel to the relayer
			mr.SubscribeToMessage(domain.StartNewRound, firstSubCh)
			mr.SubscribeToMessage(domain.ReceivedAnswer, secondSubCh)

			go mr.ReadAndRelay()

			/* actions */
			// read each message from the channel and increment the count
			wg.Add(1)
			go func() {
				for range firstSubCh {
					gotStartNewRound++
				}
				wg.Done()
			}()

			wg.Add(1)
			go func() {
				for range secondSubCh {
					gotReceivedAnswer++
				}
				wg.Done()
			}()

			// assertions
			// count of received StartNewRound and ReceivedAnswer messages should be two
			wg.Wait()
			require.Equal(t, 2, gotStartNewRound)
			require.Equal(t, 2, gotReceivedAnswer)
		}
	}

	for _, tc := range priorityMaker {
		t.Run(makeTestCaseName(tc.Name), makeTestCase(tc.Maker.(MakePriorityMessageRelayerServer)))
	}

}

func Test_multiple_subscribers_same_topic_priority(t *testing.T) {
	makeTestCaseName := func(n string) string {
		return fmt.Sprintf("%smultiple subscribers of same topic receive all messages of correct type", n)
	}

	makeTestCase := func(m MakePriorityMessageRelayerServer) func(t *testing.T) {
		return func(t *testing.T) {
			/* setup */
			var wg sync.WaitGroup
			// sub to start new round messages, bufferred because
			// busy channels get their messages dropped
			firstSubCh := make(chan domain.Message, 2)
			gotStartNewRound := 0

			secondSubCh := make(chan domain.Message, 2)
			gotReceivedAnswer := 0

			thirdSubCh := make(chan domain.Message, 2)
			gotStartNewRoundToo := 0

			// use stub socket
			responses := []network.NetworkResponse{
				StartNewRoundResponse,
				ReceivedAnswerResponse,
				NetworkErrorResponse,
				NetworkErrorResponse,
				StartNewRoundResponse,
				ReceivedAnswerResponse,
				NetworkErrorResponse,
			}
			socket := network.NewNetworkSocketStub(responses)

			mr := NewDefaultMessageRelayer(socket)

			// subscribe each channel to the relayer
			mr.SubscribeToMessage(domain.StartNewRound, firstSubCh)
			mr.SubscribeToMessage(domain.ReceivedAnswer, secondSubCh)
			mr.SubscribeToMessage(domain.StartNewRound, thirdSubCh)

			go mr.ReadAndRelay()

			/* actions */
			// read each message from the channel and increment the count
			wg.Add(1)
			go func() {
				for range firstSubCh {
					gotStartNewRound++
				}
				wg.Done()
			}()

			wg.Add(1)
			go func() {
				for range secondSubCh {
					gotReceivedAnswer++
				}
				wg.Done()
			}()

			wg.Add(1)
			go func() {
				for range thirdSubCh {
					gotStartNewRoundToo++
				}
				wg.Done()
			}()

			// assertions
			// count of received StartNewRound and ReceivedAnswer messages should be two
			wg.Wait()
			require.Equal(t, 2, gotStartNewRound)
			require.Equal(t, 2, gotStartNewRoundToo)
			require.Equal(t, 2, gotReceivedAnswer)

		}
	}

	for _, tc := range priorityMaker {
		t.Run(makeTestCaseName(tc.Name), makeTestCase(tc.Maker.(MakePriorityMessageRelayerServer)))
	}

}
