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
func Start() {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		responses   = []network.NetworkResponse{
			{
				Message: &domain.Message{
					Type: domain.ReceivedAnswer,
				},
			},
		}
		ns = network.NewNetworkSocketStub(responses)
		mr = relayer.NewDefaultMessageRelayer(ns)
	)

	go mr.Start()
	defer mr.Stop()
	defer cancel()

	l := mr.Subscribe(domain.ReceivedAnswer)

	done := utils.Take(ctx.Done(), l, 5)
	<-done
}
