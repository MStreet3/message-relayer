package mailbox

import (
	"sync"
	"time"
)

type timestamper struct {
	mu        sync.RWMutex
	timestamp int64
}

func NewTimeStamper() *timestamper {
	return &timestamper{
		mu: sync.RWMutex{},
	}
}

func (ts *timestamper) SetTimestamp() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.timestamp = time.Now().UTC().UnixNano()
}

func (ts *timestamper) GetTimestamp() int64 {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return ts.timestamp
}
