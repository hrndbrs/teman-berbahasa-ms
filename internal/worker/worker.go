package worker

import (
	"context"
	"log/slog"
)

type Job interface {
	Type() string
	Execute(ctx context.Context)
}

type Worker struct {
	jobs chan Job
}

func New(bufferSize int) *Worker {
	return &Worker{jobs: make(chan Job, bufferSize)}
}

func (w *Worker) Start(ctx context.Context, concurrency int) {
	for range concurrency {
		go func() {
			for {
				select {
				case job := <-w.jobs:
					slog.DebugContext(ctx, "executing job", "type", job.Type())
					job.Execute(ctx)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

// Dispatch is non-blocking. Drops the job if the channel is full.
func (w *Worker) Dispatch(job Job) {
	select {
	case w.jobs <- job:
		slog.Debug("job dispatched", "type", job.Type())
	default:
		slog.Warn("worker queue full, dropping job", "type", job.Type())
	}
}
