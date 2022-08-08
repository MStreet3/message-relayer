package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/mstreet3/message-relayer/app"
	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/relayer"
	"github.com/mstreet3/message-relayer/utils"
)

var responses = []network.NetworkResponse{
	{
		Message: &domain.Message{
			Type: domain.ReceivedAnswer,
		},
	},
}

func main() {
	var (
		ctxWithCancel, cancel = context.WithCancel(context.Background())
		ns                    = network.NewNetworkSocketStub(responses)
		mr                    = relayer.NewDefaultMessageRelayer(ns)
		interrupt             = make(chan os.Signal, 1)
	)

	// Notify main of any interruptions
	signal.Notify(interrupt, os.Interrupt)

	application := app.NewApplication(ns, mr)

	utils.DPrintf("starting the app")
	stopped := application.Start(ctxWithCancel)

	// Handle graceful shutdown
	for {
		select {
		case <-stopped:
			log.Println("app is stopped, goodbye")
			return

		case <-interrupt:
			log.Println("starting graceful shutdown")
			cancel()
		}
	}
}
