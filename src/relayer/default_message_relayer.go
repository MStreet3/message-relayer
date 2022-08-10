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

type messageRelayer struct {
	om      ObserverManager
	network network.RestartNetworkReader
	errorCh <-chan error
}

func NewDefaultMessageRelayer(n network.RestartNetworkReader) MessageRelayer {
	return &messageRelayer{
		network: n,
		om:      NewObserverManager(),
	}
}

func (mr *messageRelayer) Start(ctx context.Context) <-chan struct{} {
	var (
		ctxwc, cancel         = context.WithCancel(ctx)
		terminated            = make(chan struct{})
		reading, msgCh, errCh = mr.read(ctxwc)
		relaying              = mr.relay(ctxwc, msgCh)
	)

	mr.errorCh = errCh

	go func() {
		defer close(terminated)
		defer mr.om.Close()
		defer cancel()
		<-reading
		<-relaying
	}()

	return terminated
}

func (mr *messageRelayer) Subscribe(mt domain.MessageType) (<-chan domain.Message, func()) {
	return mr.om.Subscribe(context.Background(), mt)
}

func (mr *messageRelayer) Errors() <-chan error {
	return mr.errorCh
}

func (mr *messageRelayer) read(ctx context.Context) (<-chan struct{}, <-chan domain.Message, <-chan error) {
	var (
		msgCh = make(chan domain.Message)
		errCh = make(chan error)
		done  = make(chan struct{})
	)

	go func() {
		defer close(done)
		defer close(msgCh)
		defer close(errCh)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := mr.network.Read()
				if err != nil {
					select {
					case errCh <- err:
					default:
						utils.DPrintf("no error subscribers")
					}

					if errors.Is(err, errs.FatalSocketError{}) {
						utils.DPrintf("%s\n", err.Error())

						if rerr := mr.network.Restart(); rerr != nil {
							log.Fatal(rerr)
						}
					}

					continue
				}

				utils.DPrintf("relaying message of type %s\n", msg.Type)
				select {
				case msgCh <- *msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return done, msgCh, errCh
}

func (mr *messageRelayer) relay(ctx context.Context, msgCh <-chan domain.Message) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, open := <-msgCh:
				if !open {
					return
				}
				mr.om.Notify(ctx, msg)
			}
		}
	}()

	return done
}
