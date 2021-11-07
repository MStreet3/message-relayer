package domain

type MessageType int

const (
	StartNewRound MessageType = iota << 1
	ReceivedAnswer
)

/* number of messages held in memory for each
message type.  additional messages are added by dropping
the oldest message */
var PriorityQueueCapacity int = 100

type Message struct {
	Type MessageType
	Data []byte
}
