package relayer

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/stretchr/testify/require"
)

var StartNewRoundResponse = network.NetworkResponse{Message: domain.Message{
	Type: domain.StartNewRound,
	Data: nil,
},
	Error: nil,
}

var ReceivedAnswerResponse = network.NetworkResponse{Message: domain.Message{
	Type: domain.ReceivedAnswer,
	Data: nil,
},
	Error: nil,
}

var NetworkErrorResponse = network.NetworkResponse{Message: domain.Message{},
	Error: errors.New("network unavailable"),
}

type MessageRelayerServerTestCase struct {
	Name  string
	Maker MakeMessageRelayerServer
}

func Test_single_subscriber(t *testing.T) {

	makeTestCaseName := func(n string) string {
		return fmt.Sprintf("%ssubscriber receives all messages of correct type", n)
	}

	testCases := []MessageRelayerServerTestCase{
		{
			Name:  "DefaultMessageRelayer:",
			Maker: NewDefaultMessageRelayer,
		},
	}

	makeTestCase := func(m MakeMessageRelayerServer) func(t *testing.T) {
		return func(t *testing.T) {
			/* setup */
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

			go mr.ListenAndRelay()

			/* actions */
			// read each message from the channel and increment the count
			for msg := range firstSubCh {
				fmt.Printf("received message: %#v\n", msg)
				gotStartNewRound++
			}

			// assertions
			// count of received StartNewRound messages should be two
			require.Equal(t, 2, gotStartNewRound)
		}
	}

	for _, tc := range testCases {
		t.Run(makeTestCaseName(tc.Name), makeTestCase(tc.Maker))
	}

}

func Test_all_subscribers_receives_all_messages_of_correct_type(t *testing.T) {
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

	go mr.ListenAndRelay()

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

func Test_relayer_continues_with_non_fatal_network_errors(t *testing.T) {
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

	go mr.ListenAndRelay()

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

func Test_multiple_subs_of_same_type_receive_all_messages_of_correct_type(t *testing.T) {
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

	go mr.ListenAndRelay()

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
