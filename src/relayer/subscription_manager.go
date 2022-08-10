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
	Notify(ctx context.Context, msg domain.Message)
	Close()
}

type subscriptionManager struct {
	subscribers map[domain.MessageType]map[string]MessageObserver
	stopCh      chan struct{}
	mu          sync.Mutex
	wg          sync.WaitGroup
}

func NewSubscriptionManager() SubscriptionManager {
	return &subscriptionManager{
		mu:          sync.Mutex{},
		wg:          sync.WaitGroup{},
		stopCh:      make(chan struct{}),
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
		handler   = func(msg domain.Message) error {
			select {
			case <-utils.CtxOrDone(ctx, sm.stopCh):
				utils.DPrintf("%s: received stop signal", id)
			case msgCh <- msg:
				utils.DPrintf("%s: received message", id)
			default:
				utils.DPrintf("%s: dropped message", id)
			}
			return nil
		}
		mo = NewMessageObserver(id, handler)
	)

	if _, ok := sm.subscribers[mt]; !ok {
		sm.subscribers[mt] = make(map[string]MessageObserver)
	}

	sm.subscribers[mt][id.String()] = mo

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		defer close(msgCh)
		select {
		case <-utils.CtxOrDone(ctx, sm.stopCh):
			utils.DPrintf("%s: received stop signal, closing chan", id)
		case <-cleanupCh:
			utils.DPrintf("%s: received cleanup signal, closing chan", id)
		}
	}()

	return msgCh, func() {
		defer close(cleanupCh)
		sm.remove(mt, id)
	}
}

func (sm *subscriptionManager) Notify(ctx context.Context, msg domain.Message) {
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

func (sm *subscriptionManager) Close() {
	defer sm.wg.Wait()
	close(sm.stopCh)
}

func (sm *subscriptionManager) remove(mt domain.MessageType, uuid uuid.UUID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.subscribers[mt], uuid.String())
}
