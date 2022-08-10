package relayer

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/errs"
	"github.com/mstreet3/message-relayer/lruCache"
	"github.com/mstreet3/message-relayer/network"
	"github.com/mstreet3/message-relayer/utils"
)

type PriorityMessageRelayer struct {
	sm      *subscriptionManager
	network network.RestartNetworkReader
	queues  map[domain.MessageType]lruCache.PriorityQueue
	errorCh chan error
	stopCh  chan struct{}
	mu      sync.Mutex
	wg      sync.WaitGroup
}

func (mr *PriorityMessageRelayer) Subscribe(mt domain.MessageType) (<-chan domain.Message, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		<-mr.stopCh
	}()

	return mr.sm.Subscribe(ctx, mt)
}

func (mr *PriorityMessageRelayer) Start(ctx context.Context) <-chan struct{} {
	terminated := make(chan struct{})

	go func() {
		defer close(terminated)
		defer mr.sm.Close()
		defer close(mr.stopCh)

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
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		<-mr.stopCh
	}()

	for msg := range ch {
		mr.sm.Notify(ctx, msg)
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
		network: n,
		queues:  queues,
		errorCh: make(chan error),
		stopCh:  make(chan struct{}),
		mu:      sync.Mutex{},
		wg:      sync.WaitGroup{},
	}
}
