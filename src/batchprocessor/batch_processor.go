package batchprocessor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mstreet3/message-relayer/utils"
)

type Job struct {
	ID uuid.UUID
}

func NewJob() Job {
	return Job{
		ID: uuid.New(),
	}
}

type BatchProcessor interface {
	Start(context.Context) <-chan struct{}
	Stop()
	Process(Job)
	Events() <-chan string
}

type batchProcessor struct {
	size   int
	jobCh  chan Job
	stopCh chan struct{}
	events chan string
}

func NewBatchProcessor(n int) BatchProcessor {
	return &batchProcessor{
		size:   n,
		jobCh:  make(chan Job),
		stopCh: make(chan struct{}),
		events: make(chan string, 1),
	}
}

func (bp *batchProcessor) Start(ctx context.Context) <-chan struct{} {
	var (
		ctxwc, cancel = context.WithCancel(ctx)
		stopped       = make(chan struct{})
	)

	// initialize pipeline
	processing := bp.enqueue(ctxwc, bp.processBatches(ctxwc, bp.batcher(ctxwc, bp.jobCh)))

	go func() {
		defer cancel()
		defer close(bp.jobCh)
		defer utils.DPrintf("starting shutdown")
		<-bp.stopCh
	}()

	go func() {
		defer close(stopped)
		defer utils.DPrintf("batch processor shutdown complete")
		<-processing
	}()

	return stopped
}

func (bp *batchProcessor) Stop() {
	select {
	case <-bp.stopCh:
		return
	default:
		close(bp.stopCh)
	}
}

func (bp *batchProcessor) Process(j Job) {
	select {
	case <-bp.stopCh:
		return
	case bp.jobCh <- j:
		var (
			ctxwc, cancel = context.WithCancel(context.Background())
			notified      = make(chan struct{})
		)

		defer close(notified)
		go func() {
			defer cancel()
			select {
			case <-bp.stopCh:
			case <-notified:
			}
		}()

		bp.notify(ctxwc, "job sent to process")
	}
}

func (bp *batchProcessor) Events() <-chan string {
	return bp.events
}

func (bp *batchProcessor) batcher(ctx context.Context, jobs <-chan Job) <-chan []Job {
	var (
		batched = make(chan []Job)
		batch   = make([]Job, bp.size)
		i       = 0
	)

	go func() {
		defer utils.DPrintf("batcher closed")
		defer close(batched)
		for job := range jobs {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if i == bp.size-1 {
				select {
				case <-ctx.Done():
					return
				case batched <- batch:
					go bp.notify(ctx, fmt.Sprintf("batcher: sent batch %#v", batch))
				}
				i = 0
				continue
			}

			batch[i] = job
			i++
		}
	}()
	return batched
}

// processBatches processes a batch of jobs by calling runJob for each job.  any failed job runs
// are placed onto a retry channel.
func (bp *batchProcessor) processBatches(ctx context.Context, batches <-chan []Job) <-chan Job {
	retries := make(chan Job)
	go func() {
		defer utils.DPrintf("procBatch closed")
		defer close(retries)
		for batch := range batches {
			select {
			case <-ctx.Done():
				return
			default:
			}
			bp.processBatch(ctx, retries, batch)
		}
	}()
	return retries
}

func (bp *batchProcessor) processBatch(ctx context.Context, retries chan<- Job, batch []Job) {
	for _, job := range batch {
		if err := bp.runJob(ctx, job); err != nil {
			select {
			case <-ctx.Done():
				return
			case retries <- job:
			}
		}
		go bp.notify(ctx, fmt.Sprintf("procBatch: ran job %#v", job))
	}
}

func (bp *batchProcessor) runJob(ctx context.Context, job Job) error {
	<-time.After(500 * time.Millisecond)
	return nil
}

func (bp *batchProcessor) enqueue(ctx context.Context, jobs <-chan Job) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer utils.DPrintf("enqueue closed")
		defer close(done)
		for j := range jobs {
			select {
			case <-ctx.Done():
				return
			case bp.jobCh <- j:
			}
		}
	}()
	return done
}

func (bp *batchProcessor) notify(ctx context.Context, msg string) {
	select {
	case <-ctx.Done():
		return
	case bp.events <- msg:
		return
	}
}
