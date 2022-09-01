package batchprocessor

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

type Batch []Job

func (b Batch) String() string {
	out := make([]string, len(b))
	for i, job := range b {
		out[i] = job.ID.String()
	}
	return fmt.Sprintf("{ %s }", strings.Join(out, ","))
}

type BatchProcessor interface {
	Start(context.Context) <-chan struct{}
	Stop() error
	Process(Job) error
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
		ctxwc, cancel       = context.WithCancel(ctx)
		stopped             = make(chan struct{})
		batches, isBatching = bp.batcher(ctxwc, bp.jobCh)
		retries, isRetrying = bp.processBatches(ctxwc, batches)
		isProcessing        = bp.enqueue(ctxwc, retries, bp.jobCh)
	)

	// listen for shutdown signal
	go func() {
		defer cancel()
		defer utils.DPrintf("starting shutdown")
		<-bp.stopCh
	}()

	// await cleanup
	go func() {
		defer close(stopped)
		defer utils.DPrintf("batch processor shutdown complete")
		<-isBatching
		<-isRetrying
		<-isProcessing
	}()

	return stopped
}

func (bp *batchProcessor) Stop() error {
	select {
	case <-bp.stopCh:
		return errors.New("batch processor is stopped")
	default:
		close(bp.stopCh)
		return nil
	}
}

func (bp *batchProcessor) Process(j Job) error {
	select {
	case <-bp.stopCh:
		return errors.New("batch processor is stopped")
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

		bp.notify(ctxwc, fmt.Sprintf("process: job sent %s", j.ID))
		return nil
	}
}

func (bp *batchProcessor) Events() <-chan string {
	return bp.events
}

func (bp *batchProcessor) batcher(ctx context.Context, jobs <-chan Job) (<-chan []Job, <-chan struct{}) {
	var (
		done    = make(chan struct{})
		batches = make(chan []Job)
		batch   = make([]Job, bp.size)
		i       = 0
	)

	go func() {
		defer utils.DPrintf("batcher closed")
		defer close(done)
		defer close(batches)
		for {
			select {
			case <-ctx.Done():
				return
			case job, open := <-jobs:
				if !open {
					return
				}
				batch[i] = job
				if i == bp.size-1 {
					select {
					case <-ctx.Done():
						return
					case batches <- batch:
						go bp.notify(ctx, fmt.Sprintf("batcher: sent batch %s", batch))
					}
					i = 0
					continue
				}
				i++
			}
		}
	}()

	return batches, done
}

// processBatches processes a batch of jobs by calling runJob for each job.  any failed job runs
// are placed onto a retry channel.
func (bp *batchProcessor) processBatches(ctx context.Context, batches <-chan []Job) (<-chan Job, <-chan struct{}) {
	var (
		done    = make(chan struct{})
		retries = make(chan Job)
	)

	go func() {
		defer utils.DPrintf("procBatch closed")
		defer close(done)
		defer close(retries)
		for batch := range batches {
			select {
			case <-ctx.Done():
				return
			default:
			}
			go bp.processBatch(ctx, retries, batch)
		}
	}()

	return retries, done
}

func (bp *batchProcessor) processBatch(ctx context.Context, retries chan<- Job, batch []Job) {
	for _, job := range batch {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := bp.runJob(ctx, job); err != nil {
			select {
			case <-ctx.Done():
				return
			case retries <- job:
			}
		}
		go bp.notify(ctx, fmt.Sprintf("procBatch: ran job %s", job.ID))
	}
}

func (bp *batchProcessor) runJob(ctx context.Context, job Job) error {
	select {
	case <-time.After(500 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (bp *batchProcessor) enqueue(ctx context.Context, src <-chan Job, dest chan<- Job) <-chan struct{} {
	var (
		done = make(chan struct{})
	)

	go func() {
		defer utils.DPrintf("enqueue closed")
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case j, open := <-src:
				if !open {
					return
				}
				select {
				case <-ctx.Done():
					return
				case dest <- j:
				}
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
