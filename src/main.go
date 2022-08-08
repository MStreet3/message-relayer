package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/mstreet3/message-relayer/app"
	"github.com/mstreet3/message-relayer/utils"
)

func main() {
	utils.DPrintf("starting the app")

	// Notify main of any interruptions
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ctxWithCancel, cancel := context.WithCancel(context.Background())
	shutdown := app.Start(ctxWithCancel)

	// Handle graceful shutdown
	for {
		select {
		case <-shutdown:
			log.Println("shutdown complete, goodbye")
			return

		case <-interrupt:
			log.Println("starting graceful shutdown")
			cancel()
		}
	}
}
