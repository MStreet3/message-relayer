package app

import (
	"sync"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/relayer"
	"github.com/mstreet3/message-relayer/utils"
)

/*
a basic example of how a subscriber could use the
message relayer.
*/
func Start() {
	var wg sync.WaitGroup
	responses := []network.NetworkResponse{
		{Message: domain.Message{Type: domain.ReceivedAnswer}}}
	ns := network.NewNetworkSocketStub(responses)
	mr := relayer.NewDefaultMessageRelayer(ns)
	ch := make(chan domain.Message)
	mr.SubscribeToMessage(domain.ReceivedAnswer, ch)
	go mr.ReadAndRelay()

	wg.Add(1)
	go func() {
		for msg := range ch {
			utils.DPrintf("%#v", msg)
		}
		wg.Done()
	}()

	wg.Wait()

}
