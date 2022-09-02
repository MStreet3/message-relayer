package mailbox

import "context"

type Mailbox[T any] interface {
	Postmark(*T)
	Add(T)
	Empty(context.Context) <-chan T
	EmptiedAt() int64
}

type Emptier[T any] interface {
	Empty() []T
}

type Stack[T any] interface {
	Len() int
	Pop() (*T, bool)     // remove top item from stack
	PopBack() (*T, bool) // remove bottom item from stack
	PushFront(T)         // place item on top of stack
}

type TimeStamper interface {
	Timestamp() int64
}
