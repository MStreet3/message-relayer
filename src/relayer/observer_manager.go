package relayer

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/utils"
)

type ObserverManager interface {
	Subscribe(ctx context.Context, mt domain.MessageType) (<-chan domain.Message, func())
	Notify(ctx context.Context, msg domain.Message)
	Close()
}

type observerManager struct {
	subscribers map[domain.MessageType]map[string]MessageObserver
	stopCh      chan struct{}
	mu          sync.Mutex
	wg          sync.WaitGroup
}

func NewObserverManager() ObserverManager {
	return &observerManager{
		mu:          sync.Mutex{},
		wg:          sync.WaitGroup{},
		stopCh:      make(chan struct{}),
		subscribers: make(map[domain.MessageType]map[string]MessageObserver),
	}
}

func (sm *observerManager) Subscribe(ctx context.Context, mt domain.MessageType) (<-chan domain.Message, func()) {
	var (
		cleanupCh = make(chan struct{})
		unsubbed  = make(chan struct{})
		msgCh     = make(chan domain.Message, 1)
		id        = uuid.New()
		stop      = utils.CtxOrDone(ctx, sm.stopCh)
		handler   = func(msg domain.Message) error {
			select {
			case <-stop:
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

	// add message observer to subscriber map
	sm.add(mt, id, mo)

	// listen for signal to close sub channel
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		defer close(unsubbed)
		defer close(msgCh)
		select {
		case <-stop:
			utils.DPrintf("%s: received stop signal, closing chan", id)
		case <-cleanupCh:
			utils.DPrintf("%s: received cleanup signal, closing chan", id)
		}
	}()

	return msgCh, func() {
		sm.remove(mt, id)
		close(cleanupCh)
		<-unsubbed
	}
}

func (sm *observerManager) Notify(ctx context.Context, msg domain.Message) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stop := utils.CtxOrDone(ctx, sm.stopCh)
	msgType := msg.Type
	for _, sub := range sm.subscribers[msgType] {
		select {
		case <-stop:
			return
		default:
			sm.wg.Add(1)
			go func(sub MessageObserver) {
				defer sm.wg.Done()
				select {
				case <-stop:
					return
				default:
					_ = sub.Observe(msg)
				}
			}(sub)
		}
	}
}

func (sm *observerManager) Close() {
	defer sm.wg.Wait()
	close(sm.stopCh)
}

func (sm *observerManager) add(mt domain.MessageType, id uuid.UUID, mo MessageObserver) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.subscribers[mt]; !ok {
		sm.subscribers[mt] = make(map[string]MessageObserver)
	}

	sm.subscribers[mt][id.String()] = mo
}

func (sm *observerManager) remove(mt domain.MessageType, uuid uuid.UUID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.subscribers[mt], uuid.String())
}
