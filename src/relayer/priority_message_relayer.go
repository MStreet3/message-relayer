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

type PriorityQueue[T any] interface {
	Len() int
	Pop() (*T, bool)
	Push(T)
}
type priorityMessageRelayer struct {
	om      *observerManager
	network network.RestartNetworkReader
	queues  map[domain.MessageType]PriorityQueue[domain.Message]
	errorCh chan error
	stopCh  chan struct{}
	mu      sync.Mutex
	wg      sync.WaitGroup
}

func (mr *priorityMessageRelayer) Subscribe(mt domain.MessageType) (<-chan domain.Message, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		<-mr.stopCh
	}()

	return mr.om.Subscribe(ctx, mt)
}

func (mr *priorityMessageRelayer) Start(ctx context.Context) <-chan struct{} {
	terminated := make(chan struct{})

	go func() {
		defer close(terminated)
		defer mr.om.Close()
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

func (mr *priorityMessageRelayer) Enqueue(msg domain.Message) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	var q PriorityQueue[domain.Message]
	switch msg.Type {
	case domain.StartNewRound:
		q = mr.queues[domain.StartNewRound]
	case domain.ReceivedAnswer:
		q = mr.queues[domain.ReceivedAnswer]
	}
	q.Push(msg)
}

func (mr *priorityMessageRelayer) DequeueAndRelay() {
	ch := mr.Dequeue()
	mr.Broadcast(ch)
}

func (mr *priorityMessageRelayer) Broadcast(ch <-chan domain.Message) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		<-mr.stopCh
	}()

	for msg := range ch {
		mr.om.Notify(ctx, msg)
	}
}

func (mr *priorityMessageRelayer) Dequeue() <-chan domain.Message {
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

func (mr *priorityMessageRelayer) Errors() <-chan error {
	return mr.errorCh
}
