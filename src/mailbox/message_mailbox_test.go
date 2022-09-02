package mailbox

import (
	"context"
	"sync"
	"testing"

	"github.com/mstreet3/message-relayer/domain"
	"github.com/mstreet3/message-relayer/queues/lifoqueue"
	"github.com/stretchr/testify/require"
)

type testTimeStamper struct {
	mu sync.RWMutex
	ts int64
}

func (ts *testTimeStamper) Timestamp() int64 {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.ts++
	return ts.ts
}

func TestMessageMailbox(t *testing.T) {
	tests := []struct {
		name    string
		mailbox *MessageMailbox
		helper  func(t *testing.T, m *MessageMailbox)
	}{
		{
			name: "successfully adds messages and drops oldest messsage when at capacity",
			mailbox: func() *MessageMailbox {
				stack := lifoqueue.NewLIFOQueue[domain.Message]()
				mailbox := &MessageMailbox{
					cap:         1,
					stack:       stack,
					emptier:     stack,
					timestamper: &testTimeStamper{},
				}
				return mailbox
			}(),
			helper: testMessageMailbox_Add,
		},
		{
			name: "will not add a message older than emptiedAt",
			mailbox: func() *MessageMailbox {
				stack := lifoqueue.NewLIFOQueue[domain.Message]()
				mailbox := &MessageMailbox{
					cap:         1,
					stack:       stack,
					emptier:     stack,
					timestamper: &testTimeStamper{},
				}
				return mailbox
			}(),
			helper: testMessageMailbox_Empty,
		},
		{
			name: "empties completely and updates emptiedAt",
			mailbox: func() *MessageMailbox {
				stack := lifoqueue.NewLIFOQueue[domain.Message]()
				mailbox := &MessageMailbox{
					cap:         2,
					stack:       stack,
					emptier:     stack,
					timestamper: &testTimeStamper{},
				}
				return mailbox
			}(),
			helper: testMessageMailbox_EmptiedAt,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			tc.helper(t, tc.mailbox)
		})
	}
}

func testMessageMailbox_Add(t *testing.T, mailbox *MessageMailbox) {
	t.Helper()
	first := domain.NewMessage(domain.StartNewRound, nil)
	first.Timestamp = 1
	next := domain.NewMessage(domain.StartNewRound, nil)
	next.Timestamp = 0
	mailbox.Add(first)
	mailbox.Add(next)
	got, ok := mailbox.stack.Pop()
	require.True(t, ok)
	require.Equal(t, first, *got)
}

func testMessageMailbox_Empty(t *testing.T, mailbox *MessageMailbox) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first := domain.NewMessage(domain.StartNewRound, nil)
	first.Timestamp = 0

	next := domain.NewMessage(domain.ReceivedAnswer, nil)
	next.Timestamp = 0

	mailbox.Add(first)

	_ = mailbox.Empty(ctx)

	mailbox.Add(next)

	gotMsg, open := <-mailbox.Empty(ctx)
	require.Equal(t, domain.Message{}, gotMsg)
	require.True(t, !open)
}

func testMessageMailbox_EmptiedAt(t *testing.T, mailbox *MessageMailbox) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first := domain.NewMessage(domain.StartNewRound, nil)
	first.Timestamp = 0

	next := domain.NewMessage(domain.ReceivedAnswer, nil)
	next.Timestamp = 0

	mailbox.Add(first)
	mailbox.Add(next)

	msgs := mailbox.Empty(ctx)

	gotMsg := <-msgs
	require.Equal(t, next, gotMsg)

	gotMsg = <-msgs
	require.Equal(t, first, gotMsg)

	require.Equal(t, mailbox.EmptiedAt(), int64(1))
	require.Equal(t, mailbox.stack.Len(), 0)
}
