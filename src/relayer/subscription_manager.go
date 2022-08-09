package relayer

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/utils"
)

type SubscriptionManager interface {
	Subscribe(ctx context.Context, mt domain.MessageType) (<-chan domain.Message, func())
	Relay(ctx context.Context, msg domain.Message)
	Wait()
}

type subscriptionManager struct {
	subscribers map[domain.MessageType]map[string]MessageObserver
	mu          sync.Mutex
	wg          sync.WaitGroup
}

func NewSubscriptionManager() SubscriptionManager {
	return &subscriptionManager{
		mu:          sync.Mutex{},
		wg:          sync.WaitGroup{},
		subscribers: make(map[domain.MessageType]map[string]MessageObserver),
	}
}

func (sm *subscriptionManager) Subscribe(ctx context.Context, mt domain.MessageType) (<-chan domain.Message, func()) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var (
		cleanupCh = make(chan struct{})
		msgCh     = make(chan domain.Message, 1)
		id        = uuid.New()
		handler   = func(msg domain.Message) {
			select {
			case <-ctx.Done():
				utils.DPrintf("%s: received stop signal", id)
			case msgCh <- msg:
				utils.DPrintf("%s: received message", id)
			default:
				utils.DPrintf("%s: dropped message", id)
			}
		}
		sa = NewSubscriberAddress(id, handler)
	)

	if _, ok := sm.subscribers[mt]; !ok {
		sm.subscribers[mt] = make(map[string]MessageObserver)
	}

	sm.subscribers[mt][id.String()] = sa

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		defer close(msgCh)
		select {
		case <-ctx.Done():
			utils.DPrintf("%s: received stop signal, closing chan", id)
		case <-cleanupCh:
			utils.DPrintf("%s: received cleanup signal, closing chan", id)
		}
	}()

	return msgCh, func() {
		defer close(cleanupCh)
		delete(sm.subscribers[mt], id.String())
	}
}

func (sm *subscriptionManager) Relay(ctx context.Context, msg domain.Message) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	msgType := msg.Type
	for _, sub := range sm.subscribers[msgType] {
		select {
		case <-ctx.Done():
			return
		default:
			sm.wg.Add(1)
			go func(sub MessageObserver) {
				defer sm.wg.Done()
				select {
				case <-ctx.Done():
					return
				default:
					_ = sub.Observe(msg)
				}
			}(sub)
		}
	}
}

func (sm *subscriptionManager) Wait() {
	sm.wg.Wait()
}
