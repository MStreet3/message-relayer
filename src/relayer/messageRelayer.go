package relayer

import (
	"errors"
	"fmt"
	"sync"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/errs"
	"github.com/mstreet3/message-relayer/network"
)

type MessageRelayer interface {
	SubscribeToMessage(msgType domain.MessageType, ch chan<- domain.Message)
}

type DefaultMessageRelayer struct {
	network     network.NetworkSocket
	subscribers map[domain.MessageType][]chan<- domain.Message
	errorCh     chan error
	closed      bool
	mu          sync.RWMutex
	wg          sync.WaitGroup
}

func (mr *DefaultMessageRelayer) SubscribeToMessage(msgType domain.MessageType, ch chan<- domain.Message) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.subscribers[msgType] = append(mr.subscribers[msgType], ch)
}

func (mr *DefaultMessageRelayer) ListenAndRelay() {
	defer mr.Close()
	for {
		if msg, err := mr.network.Read(); err != nil {
			if errors.Is(err, errs.FatalSocketError{}) {
				fmt.Printf("%s\n", err.Error())
				// todo: probably want to cancel all open go routines
				// by sending on a done channel
				break
			}
			select {
			case mr.errorCh <- err:
			default:
				fmt.Println("no error subscribers")
			}
		} else {
			fmt.Printf("relaying the message %#v\n", msg)
			mr.Relay(msg)
		}
	}
	mr.wg.Wait()
}

func (mr *DefaultMessageRelayer) Close() {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if !mr.closed {
		mr.closed = true
		for msgType := range mr.subscribers {
			for _, ch := range mr.subscribers[msgType] {
				close(ch)
			}
		}
		close(mr.errorCh)
	}
}

func (mr *DefaultMessageRelayer) Relay(msg domain.Message) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	msgType := msg.Type
	for _, ch := range mr.subscribers[msgType] {
		mr.wg.Add(1)
		go func(ch chan<- domain.Message) {
			select {
			case ch <- msg:
			default:
				fmt.Printf("skipping busy channel\n")
			}
			mr.wg.Done()
		}(ch)
	}
}

func NewMessageRelayer(n network.NetworkSocket) DefaultMessageRelayer {
	return DefaultMessageRelayer{
		network:     n,
		subscribers: make(map[domain.MessageType][]chan<- domain.Message),
		errorCh:     make(chan error),
	}
}
