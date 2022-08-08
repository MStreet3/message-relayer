package app

import (
	"context"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/relayer"
	"github.com/mstreet3/message-relayer/utils"
)

type Application struct {
	network  network.RestartNetworkReader
	relayer  relayer.MessageRelayerServer
	services []<-chan struct{}
}

func NewApplication(
	n network.RestartNetworkReader,
	r relayer.MessageRelayerServer,
) *Application {
	return &Application{
		network:  n,
		relayer:  r,
		services: make([]<-chan struct{}, 0),
	}
}

/*
a basic example of how a subscriber could use the
message relayer.
*/
func (a *Application) Start(ctx context.Context) <-chan struct{} {
	var (
		ctxwc, cancel = context.WithCancel(ctx)
		terminated    = make(chan struct{})
		stopped       = a.relayer.Start(ctx)
	)

	go func() {
		defer cancel()
		a.listen(ctxwc)
	}()

	go func() {
		defer close(terminated)
		<-stopped
	}()

	return terminated
}

func (a *Application) listen(ctx context.Context) {
	var (
		l    = a.relayer.Subscribe(ctx, domain.ReceivedAnswer)
		done = utils.Take(ctx.Done(), l, 5)
	)

	<-done
}
