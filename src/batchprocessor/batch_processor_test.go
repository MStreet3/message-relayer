package batchprocessor

import (
	"context"
	"strings"
	"testing"

	"github.com/mstreet3/message-relayer/utils"
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
				if strings.Contains(e, "procBatch: ran job") {
					bp.Stop()
				}
			}
		}
	}()

	bp.Process(job)
	<-stopped
}
