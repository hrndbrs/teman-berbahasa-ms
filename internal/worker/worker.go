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
		w.startWorker(ctx)
	}
}

func (w *Worker) startWorker(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "worker goroutine panicked, restarting", "panic", r)
				w.startWorker(ctx)
			}
		}()
		for {
			select {
			case job := <-w.jobs:
				slog.DebugContext(ctx, "executing job", "type", job.Type())
				w.executeJob(ctx, job)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (w *Worker) executeJob(ctx context.Context, job Job) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "job panicked", "type", job.Type(), "panic", r)
		}
	}()
	job.Execute(ctx)
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
