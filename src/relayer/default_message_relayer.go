package relayer

import (
	"context"
	"errors"
	"log"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/errs"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/utils"
)

type DefaultMessageRelayer struct {
	sm      SubscriptionManager
	network network.RestartNetworkReader
	errorCh chan error
	stopCh  chan struct{}
}

func (mr *DefaultMessageRelayer) Subscribe(mt domain.MessageType) (<-chan domain.Message, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		<-mr.stopCh
	}()

	return mr.sm.Subscribe(ctx, mt)
}

func (mr *DefaultMessageRelayer) Errors() <-chan error {
	return mr.errorCh
}

func (mr *DefaultMessageRelayer) Start(ctx context.Context) <-chan struct{} {
	terminated := make(chan struct{})

	go func() {
		defer close(terminated)
		defer mr.sm.Wait()
		defer close(mr.stopCh)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := mr.network.Read()
				if err != nil {
					select {
					case mr.errorCh <- err:
					default:
						utils.DPrintf("no error subscribers")
					}

					if errors.Is(err, errs.FatalSocketError{}) {
						utils.DPrintf("%s\n", err.Error())
						if err := mr.network.Restart(); err != nil {
							log.Fatal(err)
						}
					}

					continue
				}

				utils.DPrintf("relaying message of type %s\n", msg.Type)
				mr.Relay(ctx, *msg)
			}
		}
	}()

	return terminated
}

func (mr *DefaultMessageRelayer) Relay(ctx context.Context, msg domain.Message) {
	ctxwc, cancel := context.WithCancel(ctx)

	go func() {
		defer cancel()
		<-utils.CtxOrDone(ctxwc, mr.stopCh)
	}()

	mr.sm.Relay(ctxwc, msg)
}

func NewDefaultMessageRelayer(n network.RestartNetworkReader) MessageRelayerServer {
	return &DefaultMessageRelayer{
		network: n,
		sm:      NewSubscriptionManager(),
		errorCh: make(chan error),
		stopCh:  make(chan struct{}),
	}
}
