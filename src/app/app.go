package app

import (
	"errors"
	"fmt"
	"sync"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/errs"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/relayer"
	"github.com/mstreet3/message-relayer/utils"
)

/*
a basic example of how a subscriber could use the
message relayer.
*/
func Start() {
	var (
		wg        sync.WaitGroup
		responses = []network.NetworkResponse{
			{
				Message: &domain.Message{
					Type: domain.ReceivedAnswer,
				},
			},
		}
		errCount = 0
		msgCount = 0
		ns       = network.NewNetworkSocketStub(responses)
		mr       = relayer.NewDefaultMessageRelayer(ns)
	)

	go mr.Start()
	defer mr.Stop()

	errCh := mr.Errors()
	l := mr.Subscribe(domain.ReceivedAnswer)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case err, open := <-errCh:
				if !open {
					return
				}
				if errors.Is(err, errs.FatalSocketError{}) {
					errCount++
				}
			case <-l:
				if msgCount == 5 {
					return
				}
				msgCount++
			}
		}
	}()
	wg.Wait()

	utils.DPrintf(fmt.Sprintf("Received %d fatal errors and %d messages\n", errCount, msgCount))
}
