package mailbox

import (
	"time"
)

type timestamper struct {
}

func NewTimeStamper() *timestamper {
	return &timestamper{}
}

func (ts *timestamper) Timestamp() int64 {
	return time.Now().UTC().UnixNano()
}
