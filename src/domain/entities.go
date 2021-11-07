package domain

type MessageType int

const (
	StartNewRound MessageType = iota << 1
	ReceivedAnswer
)

var PriorityQueueCapacity int = 100

type Message struct {
	Type MessageType
	Data []byte
}
