package relayer

import (
	"context"
	"errors"
	"testing"

	"github.com/mstreet3/message-relayer/domain"
	queue "github.com/mstreet3/message-relayer/mailbox"
	"github.com/mstreet3/message-relayer/network"
	lfq "github.com/mstreet3/message-relayer/queues/lifoqueue"
	"github.com/mstreet3/message-relayer/utils"
	"github.com/stretchr/testify/require"
)

var (
	emptySNR              domain.Message = domain.NewMessage(domain.StartNewRound, nil)
	emptyRA               domain.Message = domain.NewMessage(domain.ReceivedAnswer, nil)
	StartNewRoundResponse                = network.NetworkResponse{
		Message: &emptySNR,
		Error:   nil,
	}
	ReceivedAnswerResponse = network.NetworkResponse{
		Message: &emptyRA,
		Error:   nil,
	}
	NetworkErrorResponse = network.NetworkResponse{
		Error: errors.New("network unavailable"),
	}
)

func Test_MessageRelayer_RelaysMessages(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		responses   = []network.NetworkResponse{
			StartNewRoundResponse,
			NetworkErrorResponse,
			StartNewRoundResponse,
			ReceivedAnswerResponse,
			NetworkErrorResponse,
		}
		socket = network.NewNetworkSocketStub(responses)

		lifo = lfq.NewLIFOQueue[domain.Message]()

		mr = NewMessageRelayer(
			socket,
			queue.NewMessageMailbox(1, lifo, lifo),
			NewMessageObserverManager(),
		)
		wantSNR = 6
		wantRA  = 3
		gotSNR  = 0
		gotRA   = 0
	)

	terminated := mr.Start(ctx)

	// subscribe to the relayer
	snrCh, _ := mr.Subscribe(domain.StartNewRound)
	raCh, _ := mr.Subscribe(domain.ReceivedAnswer)

	// read from the subscribers
	takeSNR := utils.TakeN(ctx.Done(), snrCh, wantSNR)
	takeRA := utils.TakeN(ctx.Done(), raCh, wantRA)
	doneSNR := make(chan struct{})
	doneRA := make(chan struct{})

	go func() {
		defer close(doneSNR)
		for range takeSNR {
			gotSNR++
		}
	}()

	go func() {
		defer close(doneRA)
		for range takeRA {
			gotRA++
		}
	}()

	// stop relayer when all done
	go func() {
		defer cancel()
		<-doneSNR
		<-doneRA
	}()

	<-terminated

	// assertions
	require.Equal(t, wantSNR, gotSNR)
	require.Equal(t, wantRA, gotRA)

	_, open := <-raCh
	require.False(t, open)

	_, open = <-snrCh
	require.False(t, open)
}
