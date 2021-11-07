package relayer

import (
	"errors"
	"sync"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/errs"
	"github.com/mstreet3/message-relayer/lruCache"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/utils"
)

type PriorityMessageRelayer struct {
	network     network.NetworkSocket
	subscribers map[domain.MessageType][]chan<- domain.Message
	queues      map[domain.MessageType]lruCache.MessagePriorityQueue
	errorCh     chan error
	closed      bool
	mu          *sync.RWMutex
	wg          *sync.WaitGroup
}

func (mr PriorityMessageRelayer) SubscribeToMessage(msgType domain.MessageType, ch chan<- domain.Message) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.subscribers[msgType] = append(mr.subscribers[msgType], ch)
}

func (mr PriorityMessageRelayer) ReadAndRelay() {
	defer mr.Close()
	for {
		if msg, err := mr.network.Read(); err != nil {
			select {
			case mr.errorCh <- err:
			default:
				utils.DPrintf("no error subscribers")
			}
			if errors.Is(err, errs.FatalSocketError{}) {
				utils.DPrintf("%s\n", err.Error())
				// todo: probably want to cancel all open go routines
				// by sending on a done channel
				break
			}
		} else {
			utils.DPrintf("relaying the message %#v\n", msg)
			mr.Relay(msg)
		}
	}
	mr.wg.Wait()
}

func (mr PriorityMessageRelayer) Enqueue(msg domain.Message) error {
	return nil
}

func (mr PriorityMessageRelayer) DequeueAndRelay() {

}

func (mr PriorityMessageRelayer) Broadcast(ch chan domain.Message) {

}

func (mr PriorityMessageRelayer) Dequeue() chan domain.Message {
	return nil
}

func (mr PriorityMessageRelayer) Len(msgType domain.MessageType) int {
	return mr.queues[msgType].Queue.Length
}

func (mr *PriorityMessageRelayer) Close() {
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

func (mr *PriorityMessageRelayer) Relay(msg domain.Message) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	msgType := msg.Type
	for _, ch := range mr.subscribers[msgType] {
		mr.wg.Add(1)
		go func(ch chan<- domain.Message) {
			select {
			case ch <- msg:
			default:
				utils.DPrintf("skipping busy channel\n")
			}
			mr.wg.Done()
		}(ch)
	}
}

func NewPriorityMessageRelayer(n network.NetworkSocket) PriorityMessageRelayerServer {

	// initialize priority queues
	// TODO: read queue capacity from config file
	queues := make(map[domain.MessageType]lruCache.MessagePriorityQueue)
	msgTypes := []domain.MessageType{domain.StartNewRound, domain.ReceivedAnswer}
	for _, t := range msgTypes {
		queues[t] = lruCache.MessagePriorityQueue{Capacity: domain.PriorityQueueCapacity}
	}

	return PriorityMessageRelayer{
		network:     n,
		subscribers: make(map[domain.MessageType][]chan<- domain.Message),
		queues:      queues,
		errorCh:     make(chan error),
		mu:          &sync.RWMutex{},
		wg:          &sync.WaitGroup{},
	}
}
