package domain

import "time"

type MessageType int

const (
	StartNewRound MessageType = iota << 1
	ReceivedAnswer
)

func (m MessageType) String() string {
	switch m {
	case StartNewRound:
		return "StartNewRound"
	case ReceivedAnswer:
		return "ReceivedAnswer"
	default:
		return "Unknown"
	}
}

/* number of messages held in memory for each
message type.  additional messages are added by dropping
the oldest message */
var PriorityQueueCapacity int = 100

type Message struct {
	Type      MessageType
	Data      []byte
	Timestamp time.Time
}
