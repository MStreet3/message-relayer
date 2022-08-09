package relayer

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/errs"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/utils"
)

type DefaultMessageRelayer struct {
	network     network.RestartNetworkReader
	subscribers map[domain.MessageType]map[string]MessageObserver
	errorCh     chan error
	stopCh      chan struct{}
	mu          *sync.RWMutex
	wg          *sync.WaitGroup
}

func (mr *DefaultMessageRelayer) Subscribe(mt domain.MessageType) (<-chan domain.Message, func()) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	var (
		cleanupCh = make(chan struct{})
		msgCh     = make(chan domain.Message, 1)
		id        = uuid.New()
		handler   = func(msg domain.Message) {
			select {
			case <-mr.stopCh:
				utils.DPrintf("%s: received stop signal", id)
			case msgCh <- msg:
				utils.DPrintf("%s: received message", id)
			default:
				utils.DPrintf("%s: dropped message", id)
			}
		}
		sa = NewSubscriberAddress(id, handler)
	)

	if _, ok := mr.subscribers[mt]; !ok {
		mr.subscribers[mt] = make(map[string]MessageObserver)
	}

	mr.subscribers[mt][id.String()] = sa

	mr.wg.Add(1)
	go func() {
		defer mr.wg.Done()
		defer close(msgCh)
		select {
		case <-mr.stopCh:
			utils.DPrintf("%s: received stop signal, closing chan", id)
		case <-cleanupCh:
			utils.DPrintf("%s: received cleanup signal, closing chan", id)
		}
	}()

	return msgCh, func() {
		defer close(cleanupCh)
		delete(mr.subscribers[mt], id.String())
	}
}

func (mr *DefaultMessageRelayer) Errors() <-chan error {
	return mr.errorCh
}

func (mr *DefaultMessageRelayer) Start(ctx context.Context) <-chan struct{} {
	terminated := make(chan struct{})

	go func() {
		defer close(terminated)
		defer mr.wg.Wait()
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
				mr.Relay(ctx.Done(), *msg)
			}
		}
	}()

	return terminated
}

func (mr *DefaultMessageRelayer) Relay(stop <-chan struct{}, msg domain.Message) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	msgType := msg.Type
	for _, sub := range mr.subscribers[msgType] {
		select {
		case <-stop:
			return
		default:
			mr.wg.Add(1)
			go func(sub MessageObserver) {
				defer mr.wg.Done()
				select {
				case <-stop:
					return
				default:
					_ = sub.Observe(msg)
				}
			}(sub)
		}
	}
}

func NewDefaultMessageRelayer(n network.RestartNetworkReader) MessageRelayerServer {
	return &DefaultMessageRelayer{
		network:     n,
		subscribers: make(map[domain.MessageType]map[string]MessageObserver),
		errorCh:     make(chan error),
		stopCh:      make(chan struct{}),
		mu:          &sync.RWMutex{},
		wg:          &sync.WaitGroup{},
	}
}
