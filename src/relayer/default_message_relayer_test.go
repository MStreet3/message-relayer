package relayer

import (
	"context"
	"testing"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
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

		mr      = NewDefaultMessageRelayer(socket)
		wantSNR = 2
		wantRA  = 2
		gotSNR  = 0
		gotRA   = 0
	)

	terminated := mr.Start(ctx)

	// subscribe to the relayer
	snrCh, _ := mr.Subscribe(domain.StartNewRound)
	raCh, _ := mr.Subscribe(domain.ReceivedAnswer)

	/* actions */
	go func() {
		defer cancel()
		for {
			select {
			case <-snrCh:
				if gotSNR < wantSNR {
					gotSNR++
				}
			case <-raCh:
				if gotRA < wantRA {
					gotRA++
				}
			}
			if gotSNR == wantSNR && gotRA == wantRA {
				return
			}
		}
	}()

	<-terminated

	// assertions
	require.Equal(t, wantSNR, gotSNR)
	require.Equal(t, wantRA, gotRA)

	_, open := <-snrCh
	require.False(t, open)

	_, open = <-raCh
	require.False(t, open)

	_, open = <-terminated
	require.False(t, open)
}
