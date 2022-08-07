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
	network     network.RestartNetworkReader
	subscribers map[domain.MessageType][]*SubscriberAddress
	queues      map[domain.MessageType]lruCache.PriorityQueue
	errorCh     chan error
	stopCh      chan struct{}
	mu          *sync.RWMutex
	wg          *sync.WaitGroup
}

func (mr *PriorityMessageRelayer) Subscribe(mt domain.MessageType) <-chan domain.Message {
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

func (mr *PriorityMessageRelayer) Start() {
	for {

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

		utils.DPrintf("relaying the message %#v\n", msg)

		mr.Enqueue(*msg)

		mr.wg.Add(1)
		go func() {
			defer mr.wg.Done()
			mr.DequeueAndRelay()
		}()

	}
}

func (mr *PriorityMessageRelayer) Enqueue(msg domain.Message) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	var q lruCache.PriorityQueue
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

func (mr *PriorityMessageRelayer) Stop() {
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

func (mr *PriorityMessageRelayer) Relay(msg domain.Message) {
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

func (mr *PriorityMessageRelayer) Errors() <-chan error {
	return mr.errorCh
}

func NewPriorityMessageRelayer(n network.RestartNetworkReader) PriorityMessageRelayerServer {
	queues := make(map[domain.MessageType]lruCache.PriorityQueue)
	msgTypes := []domain.MessageType{domain.StartNewRound, domain.ReceivedAnswer}
	for _, t := range msgTypes {
		queues[t] = lruCache.NewMessagePriorityQueue(domain.PriorityQueueCapacity)
	}

	return &PriorityMessageRelayer{
		network:     n,
		subscribers: make(map[domain.MessageType][]*SubscriberAddress),
		queues:      queues,
		errorCh:     make(chan error),
		stopCh:      make(chan struct{}),
		mu:          &sync.RWMutex{},
		wg:          &sync.WaitGroup{},
	}
}
