package domain

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

/*
	number of messages held in memory for each

message type.  additional messages are added by dropping
the oldest message
*/
var PriorityQueueCapacity int = 100

type Message struct {
	msgType   MessageType
	Data      []byte
	Timestamp int64
}

func NewMessage(t MessageType, d []byte) Message {
	return Message{
		msgType: t,
		Data:    d,
	}
}

func (m Message) Type() MessageType {
	return m.msgType
}
