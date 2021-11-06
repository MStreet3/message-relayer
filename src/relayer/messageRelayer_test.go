package relayer

import (
	"errors"
	"fmt"
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

func Test_subscriber_receives_all_messages_of_correct_type(t *testing.T) {
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

	mr := NewMessageRelayer(socket)

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
