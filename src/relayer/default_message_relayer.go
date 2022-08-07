package relayer

import (
	"errors"
	"sync"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/errs"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/utils"
)

type DefaultMessageRelayer struct {
	network     network.RestartNetworkReader
	subscribers map[domain.MessageType][]*SubscriberAddress
	errorCh     chan error
	stopCh      chan struct{}
	mu          *sync.RWMutex
	wg          *sync.WaitGroup
}

func (mr *DefaultMessageRelayer) Subscribe(mt domain.MessageType) <-chan domain.Message {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	msgCh := make(chan domain.Message)
	sa := NewSubscriberAddress(msgCh)
	mr.subscribers[mt] = append(mr.subscribers[mt], sa)

	mr.wg.Add(1)
	go func() {
		defer mr.wg.Done()
		defer close(msgCh)
		<-mr.stopCh
	}()

	return msgCh
}

func (mr *DefaultMessageRelayer) Errors() <-chan error {
	return mr.errorCh
}

func (mr *DefaultMessageRelayer) Start() {
	for {
		select {
		case <-mr.stopCh:
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
					mr.network.Restart()
				}

				continue
			}

			utils.DPrintf("relaying message of type %s\n", msg.Type)
			mr.Relay(*msg)
		}
	}
}

func (mr *DefaultMessageRelayer) Relay(msg domain.Message) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	msgType := msg.Type
	for _, sub := range mr.subscribers[msgType] {
		select {
		case <-mr.stopCh:
			return
		default:
			mr.wg.Add(1)
			go func(sub *SubscriberAddress) {
				defer mr.wg.Done()
				select {
				case sub.msgCh <- msg:
					utils.DPrintf("%s received message of type %s\n", sub.ID.String(), msg.Type)
				case <-mr.stopCh:
				default:
					utils.DPrintf("%s is busy, dropping message of type %s\n", sub.ID.String(), msg.Type)
				}
			}(sub)
		}
	}
}

func (mr *DefaultMessageRelayer) Stop() {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	select {
	case <-mr.stopCh:
		return
	default:
		utils.DPrintf("starting graceful shutdown\n")
		close(mr.stopCh)
		utils.DPrintf("waiting on children routines\n")
		mr.wg.Wait()
		close(mr.errorCh)
		utils.DPrintf("relayer is closed\n")
	}
}

func NewDefaultMessageRelayer(n network.RestartNetworkReader) MessageRelayerServer {
	return &DefaultMessageRelayer{
		network:     n,
		subscribers: make(map[domain.MessageType][]*SubscriberAddress),
		errorCh:     make(chan error),
		stopCh:      make(chan struct{}),
		mu:          &sync.RWMutex{},
		wg:          &sync.WaitGroup{},
	}
}
