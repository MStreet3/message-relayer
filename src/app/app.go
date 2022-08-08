package app

import (
	"context"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/relayer"
	"github.com/mstreet3/message-relayer/utils"
)

/*
a basic example of how a subscriber could use the
message relayer.
*/
func Start(ctx context.Context) <-chan struct{} {
	terminated := make(chan struct{})

	go func() {
		defer close(terminated)
		run(ctx)
	}()

	return terminated
}

func run(ctx context.Context) {
	var (
		ctxwc, cancel = context.WithCancel(ctx)
		responses     = []network.NetworkResponse{
			{
				Message: &domain.Message{
					Type: domain.ReceivedAnswer,
				},
			},
		}
		ns = network.NewNetworkSocketStub(responses)
		mr = relayer.NewDefaultMessageRelayer(ns)
	)

	stopped := mr.Start(ctxwc)

	l := mr.Subscribe(ctxwc, domain.ReceivedAnswer)

	done := utils.Take(ctx.Done(), l, 500)
	<-done
	cancel()
	<-stopped
}
