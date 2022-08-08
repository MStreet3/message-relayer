package relayer

import (
	"context"
	"sync"
	"testing"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/stretchr/testify/require"
)

func Test_DefaultMessageRelayer_RelaysMessages(t *testing.T) {
	var (
		wg          sync.WaitGroup
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
	snrCh := mr.Subscribe(ctx, domain.StartNewRound)
	raCh := mr.Subscribe(ctx, domain.ReceivedAnswer)

	/* actions */
	wg.Add(1)
	go func() {
		defer wg.Done()
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
	wg.Wait()

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
