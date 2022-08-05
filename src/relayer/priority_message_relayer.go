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
	queues      map[domain.MessageType]*lruCache.MessagePriorityQueue
	errorCh     chan error
	closed      bool
	mu          *sync.RWMutex
	wg          *sync.WaitGroup
}

func (mr *PriorityMessageRelayer) Subscribe(mt domain.MessageType) <-chan domain.Message { return nil }

func (mr *PriorityMessageRelayer) SubscribeToMessage(msgType domain.MessageType, ch chan<- domain.Message) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.subscribers[msgType] = append(mr.subscribers[msgType], ch)
}

func (mr *PriorityMessageRelayer) Start() {
	defer mr.Stop()
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
			mr.wg.Add(1)
			go func(msg domain.Message) {
				defer mr.wg.Done()
				mr.Enqueue(msg)
			}(*msg)

			mr.wg.Add(1)
			go func() {
				defer mr.wg.Done()
				mr.DequeueAndRelay()
			}()
		}

	}
	mr.wg.Wait()
}

func (mr *PriorityMessageRelayer) Enqueue(msg domain.Message) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	var q *lruCache.MessagePriorityQueue
	switch msg.Type {
	case domain.StartNewRound:
		q = mr.queues[domain.StartNewRound]
	case domain.ReceivedAnswer:
		q = mr.queues[domain.ReceivedAnswer]
	}
	q.Push(msg)
}

func (mr *PriorityMessageRelayer) DequeueAndRelay() {

	ch := mr.Dequeue()
	mr.Broadcast(ch)
}

func (mr *PriorityMessageRelayer) Broadcast(ch <-chan domain.Message) {
	for msg := range ch {
		mr.Relay(msg)
	}
}

func (mr *PriorityMessageRelayer) Dequeue() <-chan domain.Message {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	sendCh := make(chan domain.Message, 3)
	go func() {
		defer close(sendCh)
		i := 0
		startNewRoundQueue := mr.queues[domain.StartNewRound]
		receivedAnswerQueue := mr.queues[domain.ReceivedAnswer]
		for msg, ok := startNewRoundQueue.Pop(); ok && i < 2; i++ {
			sendCh <- *msg
		}
		if msg, ok := receivedAnswerQueue.Pop(); ok {
			sendCh <- *msg
		}
	}()

	return sendCh
}

func (mr *PriorityMessageRelayer) Len(msgType domain.MessageType) int {
	return mr.queues[msgType].Queue.Length
}

func (mr *PriorityMessageRelayer) Stop() {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if !mr.closed {
		utils.DPrintf("starting shutdown procedure")
		mr.closed = true
		for msgType := range mr.subscribers {
			for _, ch := range mr.subscribers[msgType] {
				close(ch)
			}
		}

		utils.DPrintf("everything is shutdown")
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
	queues := make(map[domain.MessageType]*lruCache.MessagePriorityQueue)
	msgTypes := []domain.MessageType{domain.StartNewRound, domain.ReceivedAnswer}
	for _, t := range msgTypes {
		queues[t] = &lruCache.MessagePriorityQueue{Capacity: domain.PriorityQueueCapacity}
	}

	return &PriorityMessageRelayer{
		network:     n,
		subscribers: make(map[domain.MessageType][]chan<- domain.Message),
		queues:      queues,
		errorCh:     make(chan error),
		mu:          &sync.RWMutex{},
		wg:          &sync.WaitGroup{},
	}
}
