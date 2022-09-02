package relayer

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/utils"
)

type Observer[T any] interface {
	Observe(T) error
}

type ObserverManager[T ~int, E Event[T]] interface {
	Subscribe(ctx context.Context, t T) (<-chan E, func())
	Notify(ctx context.Context, evt E)
	Close()
}

type msgObserverManager struct {
	subscribers map[domain.MessageType]map[string]MessageObserver
	stopCh      chan struct{}
	mu          sync.Mutex
	wg          sync.WaitGroup
}

func NewMessageObserverManager() *msgObserverManager {
	return &msgObserverManager{
		mu:          sync.Mutex{},
		wg:          sync.WaitGroup{},
		stopCh:      make(chan struct{}),
		subscribers: make(map[domain.MessageType]map[string]MessageObserver),
	}
}

func (mom *msgObserverManager) Subscribe(ctx context.Context, mt domain.MessageType) (<-chan domain.Message, func()) {
	var (
		cleanupCh = make(chan struct{})
		unsubbed  = make(chan struct{})
		msgCh     = make(chan domain.Message)
		id        = uuid.New()
		stop      = utils.CtxOrDone(ctx, mom.stopCh)
		handler   = func(msg domain.Message) error {
			select {
			case <-stop:
				utils.DPrintf("%s: received stop signal", id)
			case msgCh <- msg:
				utils.DPrintf("%s: received message of type %s", id, msg.Type())
			default:
				utils.DPrintf("%s: dropped message of type %s", id, msg.Type())
			}
			return nil
		}
		mo = NewMessageObserver(id, handler)
	)

	// add message observer to subscriber map
	mom.add(mt, id, mo)

	// listen for signal to close sub channel
	mom.wg.Add(1)
	go func() {
		defer mom.wg.Done()
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
		mom.remove(mt, id)
		close(cleanupCh)
		<-unsubbed
	}
}

func (mom *msgObserverManager) Notify(ctx context.Context, msg domain.Message) {
	mom.mu.Lock()
	defer mom.mu.Unlock()

	stop := utils.CtxOrDone(ctx, mom.stopCh)
	msgType := msg.Type()
	for _, sub := range mom.subscribers[msgType] {
		select {
		case <-stop:
			return
		default:
			mom.wg.Add(1)
			go func(sub MessageObserver) {
				defer mom.wg.Done()
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

func (mom *msgObserverManager) Close() {
	defer utils.DPrintf("observer manager is shutdown")
	defer mom.wg.Wait()
	close(mom.stopCh)
}

func (mom *msgObserverManager) add(mt domain.MessageType, id uuid.UUID, mo MessageObserver) {
	mom.mu.Lock()
	defer mom.mu.Unlock()

	if _, ok := mom.subscribers[mt]; !ok {
		mom.subscribers[mt] = make(map[string]MessageObserver)
	}

	mom.subscribers[mt][id.String()] = mo
}

func (mom *msgObserverManager) remove(mt domain.MessageType, uuid uuid.UUID) {
	mom.mu.Lock()
	defer mom.mu.Unlock()

	delete(mom.subscribers[mt], uuid.String())
}
