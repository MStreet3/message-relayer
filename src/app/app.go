package app

import (
	"context"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/relayer"
	"github.com/mstreet3/message-relayer/utils"
)

type Application struct {
	network network.RestartNetworkReader
	relayer relayer.MessageRelayer
}

func NewApplication(
	n network.RestartNetworkReader,
	r relayer.MessageRelayer,
) *Application {
	return &Application{
		network: n,
		relayer: r,
	}
}

// a basic example of how a subscriber could use the
// message relayer.
func (app *Application) Start(ctx context.Context) <-chan struct{} {
	var (
		ctxwc, cancel = context.WithCancel(ctx)
		listening     = make(chan struct{})
		n             = 5 // take 5
		relaying      = app.relayer.Start(ctxwc)
		shutdown      = make(chan struct{})
	)

	// listen for messages
	go func() {
		defer close(listening)
		ra := app.listen(ctxwc, n, domain.ReceivedAnswer)
		snr := app.listen(ctxwc, n, domain.StartNewRound)
		<-ra
		<-snr
	}()

	// handle graceful shutdown
	go func() {
		defer close(shutdown)
		<-listening
		cancel()
		<-relaying
	}()

	return shutdown
}

// listen blocks to hear n messages
func (app *Application) listen(ctx context.Context, n int, mt domain.MessageType) <-chan struct{} {
	var (
		l, cleanup = app.relayer.Subscribe(mt)
		taking     = utils.ReadN(ctx.Done(), l, n)
		done       = make(chan struct{})
	)

	// take until done
	go func() {
		defer close(done)
		defer cleanup()
		<-taking
	}()

	return done
}
