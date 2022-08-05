package relayer

import (
	"sync"
	"testing"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/stretchr/testify/require"
)

func Test_DefaultMessageRelayer_RelaysMessages(t *testing.T) {
	var (
		wg sync.WaitGroup

		responses = []network.NetworkResponse{
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

	go mr.Start()

	// subscribe to the relayer
	snrCh := mr.Subscribe(domain.StartNewRound)
	raCh := mr.Subscribe(domain.ReceivedAnswer)

	/* actions */
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer mr.Stop()
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
}
