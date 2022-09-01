package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

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
	{
		Message: &domain.Message{
			Type: domain.StartNewRound,
		},
	},
}

func main() {
	var (
		ctxWithTimeout, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		ns                     = network.NewNetworkSocketStub(responses)
		mr                     = relayer.NewDefaultMessageRelayer(ns)
		interrupt              = make(chan os.Signal, 1)
	)

	// Notify main of any interruptions
	signal.Notify(interrupt, os.Interrupt)

	application := app.NewApplication(ns, mr)

	utils.DPrintf("starting the app")
	stopped := application.Start(ctxWithTimeout)

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
