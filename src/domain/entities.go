package domain

type MessageType int

const (
	StartNewRound MessageType = iota << 1
	ReceivedAnswer
)

type Message struct {
	Type MessageType
	Data []byte
}
