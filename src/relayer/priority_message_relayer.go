package relayer

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/errs"
	"github.com/mstreet3/message-relayer/lruCache"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/utils"
)

type PriorityMessageRelayer struct {
	network     network.RestartNetworkReader
	subscribers map[domain.MessageType]map[string]MessageObserver
	queues      map[domain.MessageType]lruCache.PriorityQueue
	errorCh     chan error
	stopCh      chan struct{}
	mu          *sync.RWMutex
	wg          *sync.WaitGroup
}

func (mr *PriorityMessageRelayer) Subscribe(mt domain.MessageType) (<-chan domain.Message, func()) {
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

func (mr *PriorityMessageRelayer) Start(ctx context.Context) <-chan struct{} {
	terminated := make(chan struct{})

	go func() {
		defer close(terminated)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

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

			utils.DPrintf("relaying the message %#v\n", msg)

			mr.Enqueue(*msg)

			mr.wg.Add(1)
			go func() {
				defer mr.wg.Done()
				mr.DequeueAndRelay()
			}()

		}
	}()

	return terminated
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

func (mr *PriorityMessageRelayer) Relay(msg domain.Message) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	msgType := msg.Type
	for _, sub := range mr.subscribers[msgType] {
		select {
		case <-mr.stopCh:
			return
		default:
			_ = sub.Observe(msg)
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
		subscribers: make(map[domain.MessageType]map[string]MessageObserver),
		queues:      queues,
		errorCh:     make(chan error),
		stopCh:      make(chan struct{}),
		mu:          &sync.RWMutex{},
		wg:          &sync.WaitGroup{},
	}
}
