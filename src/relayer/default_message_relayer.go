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

type mailbox[T any] interface {
	Add(T)
	Empty(context.Context) <-chan T
}

type messageRelayer struct {
	om      ObserverManager
	network network.RestartNetworkReader
	mailbox mailbox[domain.Message]
	errorCh <-chan error
}

func NewDefaultMessageRelayer(
	n network.RestartNetworkReader,
	mailbox mailbox[domain.Message],
	om ObserverManager,
) *messageRelayer {
	return &messageRelayer{
		network: n,
		mailbox: mailbox,
		om:      om,
	}
}

func (mr *messageRelayer) Start(ctx context.Context) <-chan struct{} {
	var (
		ctxwc, cancel      = context.WithCancel(ctx)
		terminated         = make(chan struct{})
		reading, hb, errCh = mr.read(ctxwc)
		checkingMsgs       = mr.checkMail(ctxwc, hb)
	)

	mr.errorCh = errCh

	go func() {
		defer close(terminated)
		defer mr.om.Close()
		defer cancel()
		<-reading
		<-checkingMsgs
	}()

	return terminated
}

func (mr *messageRelayer) Subscribe(mt domain.MessageType) (<-chan domain.Message, func()) {
	return mr.om.Subscribe(context.Background(), mt)
}

func (mr *messageRelayer) Errors() <-chan error {
	return mr.errorCh
}

func (mr *messageRelayer) read(ctx context.Context) (<-chan struct{}, <-chan struct{}, <-chan error) {
	var (
		errCh     = make(chan error)
		done      = make(chan struct{})
		hb        = make(chan struct{}, 1)
		sendPulse = func() {
			select {
			case hb <- struct{}{}:
			default:
			}
		}
		sendErr = func(err error) {
			select {
			case errCh <- err:
			default:
				utils.DPrintf("no error subscribers")
			}
		}
	)

	go func() {
		defer close(done)
		defer close(errCh)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				sendPulse()

				msg, err := mr.network.Read()
				if err != nil {
					sendErr(err)

					if errors.Is(err, errs.FatalSocketError{}) {
						utils.DPrintf("%s\n", err.Error())

						if rerr := mr.network.Restart(); rerr != nil {
							log.Fatal(rerr)
						}
					}

					continue
				}

				utils.DPrintf("placing message of type %s in mailbox\n", msg.Type)
				mr.mailbox.Add(*msg)
			}
		}
	}()

	return done, hb, errCh
}

func (mr *messageRelayer) checkMail(ctx context.Context, hb <-chan struct{}) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-hb:
				_ = mr.relay(ctx, mr.mailbox.Empty(ctx))
			}
		}
	}()

	return done
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
