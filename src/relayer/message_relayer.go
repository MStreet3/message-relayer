package relayer

import (
	"context"
	"errors"
	"log"
	"time"

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
	om      MessageObserverManager
	network network.RestartNetworkReader
	mailbox mailbox[domain.Message]
	pulse   time.Duration
}

func NewMessageRelayer(
	n network.RestartNetworkReader,
	mailbox mailbox[domain.Message],
	om MessageObserverManager,
) *messageRelayer {
	return &messageRelayer{
		network: n,
		mailbox: mailbox,
		om:      om,
		pulse:   80 * time.Millisecond,
	}
}

func (mr *messageRelayer) Start(ctx context.Context) <-chan struct{} {
	var (
		ctxwc, cancel      = context.WithCancel(ctx)
		terminated         = make(chan struct{})
		reading, hb, errCh = mr.read(ctxwc)
		monitoring         = mr.monitor(ctxwc, hb, errCh)
	)

	go func() {
		defer close(terminated)
		defer mr.om.Close()
		defer cancel()
		<-reading
		<-monitoring
	}()

	return terminated
}

func (mr *messageRelayer) Subscribe(mt domain.MessageType) (<-chan domain.Message, func()) {
	return mr.om.Subscribe(context.Background(), mt)
}

func (mr *messageRelayer) read(ctx context.Context) (<-chan struct{}, <-chan struct{}, <-chan error) {
	var (
		ticker    = time.NewTicker(mr.pulse)
		done      = make(chan struct{})
		errCh     = make(chan error, 1)
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
		enqueue = func(msg *domain.Message) {
			utils.DPrintf("placing message of type %s in mailbox\n", msg.Type())
			msg.Timestamp = time.Now().UTC().UnixNano()
			mr.mailbox.Add(*msg)
		}
	)

	go func() {
		defer close(done)
		defer close(errCh)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				msg, err := mr.network.Read()
				if err != nil {
					sendErr(err)
					continue
				}

				enqueue(msg)

				sendPulse()
			}
		}
	}()

	return done, hb, errCh
}

func (mr *messageRelayer) monitor(ctx context.Context, hb <-chan struct{}, errCh <-chan error) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-hb:
				<-mr.notify(ctx, mr.mailbox.Empty(ctx))
			case err, open := <-errCh:
				if !open {
					return
				}
				if errors.Is(err, errs.FatalSocketError{}) {
					utils.DPrintf("%s\n", err.Error())
					if rerr := mr.network.Restart(); rerr != nil {
						log.Fatal(rerr)
					}
				}
			}
		}
	}()

	return done
}

func (mr *messageRelayer) notify(ctx context.Context, msgCh <-chan domain.Message) <-chan struct{} {
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
