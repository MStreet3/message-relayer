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
	network     network.NetworkSocket
	subscribers map[domain.MessageType][]chan<- domain.Message
	errorCh     chan error
	stopCh      chan struct{}
	mu          *sync.RWMutex
	wg          *sync.WaitGroup
}

func (mr DefaultMessageRelayer) SubscribeToMessage(msgType domain.MessageType, ch chan<- domain.Message) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.subscribers[msgType] = append(mr.subscribers[msgType], ch)
}

func (mr DefaultMessageRelayer) ReadAndRelay() {
	defer mr.Close()
	for {
		msg, err := mr.network.Read();
		if err != nil {
			select {
			case mr.errorCh <- err:
			default:
				utils.DPrintf("no error subscribers")
			}
			if errors.Is(err, errs.FatalSocketError{}) {
				utils.DPrintf("%s\n", err.Error())
				break
			}
			continue
		}

		utils.DPrintf("relaying the message %#v\n", msg)
		mr.Relay(msg)
	}
}

func (mr *DefaultMessageRelayer) Relay(msg domain.Message) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	msgType := msg.Type
	for _, ch := range mr.subscribers[msgType] {
		select {
		case <-mr.stopCh:
			return
		default:
			mr.wg.Add(1)
			go func(ch chan<- domain.Message) {
				defer mr.wg.Done()
				select {
				case ch <- msg:
				case <-mr.stopCh:
				default:
					utils.DPrintf("skipping busy channel\n")
				}
			}(ch)
		}
	}
}

func (mr *DefaultMessageRelayer) Close() {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	select{
	case <-mr.stopCh:
		return
	default:
		close(mr.stopCh)
		mr.wg.Wait()
		close(mr.errorCh)
	}
}

func NewDefaultMessageRelayer(n network.NetworkSocket) MessageRelayerServer {
	return DefaultMessageRelayer{
		network:     n,
		subscribers: make(map[domain.MessageType][]chan<- domain.Message),
		errorCh:     make(chan error),
		stopCh: 	 make(chan struct{}),
		mu:          &sync.RWMutex{},
		wg:          &sync.WaitGroup{},
	}
}
