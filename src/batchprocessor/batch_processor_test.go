package batchprocessor

import (
	"context"
	"fmt"
	"testing"

	"github.com/mstreet3/message-relayer/utils"
	"github.com/stretchr/testify/require"
)

func TestBatchProcessor(t *testing.T) {
	var (
		td, _         = t.Deadline()
		ctxwd, cancel = context.WithDeadline(context.Background(), td)
		job           = NewJob()
		bp            = NewBatchProcessor(1)
		stopped       = bp.Start(ctxwd)
		evts          = bp.Events()
	)
	defer cancel()

	go func() {
		for {
			select {
			case <-ctxwd.Done():
				return
			case e, ok := <-evts:
				if !ok {
					return
				}
				utils.DPrintf(e)
				if e == fmt.Sprintf("procBatch: ran job %s", job.ID) {
					err := bp.Stop()
					require.NoError(t, err)
				}
			}
		}
	}()

	err := bp.Process(job)
	require.NoError(t, err)
	<-stopped
}
