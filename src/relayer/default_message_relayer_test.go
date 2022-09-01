package relayer

import (
	"context"
	"testing"

	"github.com/mstreet3/message-relayer/domain"
	queue "github.com/mstreet3/message-relayer/mailbox"
	"github.com/mstreet3/message-relayer/network"
	lfq "github.com/mstreet3/message-relayer/queues/lifoqueue"
	"github.com/mstreet3/message-relayer/utils"
	"github.com/stretchr/testify/require"
)

func Test_DefaultMessageRelayer_RelaysMessages(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		responses   = []network.NetworkResponse{
			StartNewRoundResponse,
			StartNewRoundResponse,
			ReceivedAnswerResponse,
		}
		socket = network.NewNetworkSocketStub(responses)

		lifo = lfq.NewLIFOQueue[domain.Message]()

		mr = NewDefaultMessageRelayer(
			socket,
			queue.NewMessageMailbox(1, lifo, lifo),
			NewObserverManager(),
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
