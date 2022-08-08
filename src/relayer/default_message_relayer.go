package relayer

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/errs"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/utils"
)

type DefaultMessageRelayer struct {
	network     network.RestartNetworkReader
	subscribers map[domain.MessageType][]MessageObserver
	errorCh     chan error
	mu          *sync.RWMutex
	wg          *sync.WaitGroup
}

func (mr *DefaultMessageRelayer) Subscribe(ctx context.Context, mt domain.MessageType) <-chan domain.Message {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	msgCh := make(chan domain.Message)
	sa := NewSubscriberAddress(msgCh)
	mr.subscribers[mt] = append(mr.subscribers[mt], sa)

	mr.wg.Add(1)
	go func() {
		defer mr.wg.Done()
		defer close(msgCh)
		<-ctx.Done()
	}()

	return msgCh
}

func (mr *DefaultMessageRelayer) Errors() <-chan error {
	return mr.errorCh
}

func (mr *DefaultMessageRelayer) Start(ctx context.Context) <-chan struct{} {
	terminated := make(chan struct{})

	go func() {
		defer close(terminated)
		defer mr.wg.Wait()

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
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	msgType := msg.Type
	for _, sub := range mr.subscribers[msgType] {
		select {
		case <-ctx.Done():
			return
		default:
			_ = sub.Observe(ctx, msg)
		}
	}
}

/*
mr.wg.Add(1)
			go func(sub *SubscriberAddress) {
				defer mr.wg.Done()
				select {
				case sub.msgCh <- msg:
					utils.DPrintf("%s received message of type %s\n", sub.ID.String(), msg.Type)
				case <-ctx.Done():
				default:
					utils.DPrintf("%s is busy, dropping message of type %s\n", sub.ID.String(), msg.Type)
				}
			}(sub) */

func NewDefaultMessageRelayer(n network.RestartNetworkReader) MessageRelayerServer {
	return &DefaultMessageRelayer{
		network:     n,
		subscribers: make(map[domain.MessageType][]MessageObserver),
		errorCh:     make(chan error),
		mu:          &sync.RWMutex{},
		wg:          &sync.WaitGroup{},
	}
}
