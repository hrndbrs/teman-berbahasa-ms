package worker

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
)

type Job interface {
	Type() string
	Execute(ctx context.Context)
	LogFields() []any
}

type Worker struct {
	jobs         chan Job
	wg           sync.WaitGroup
	droppedTotal atomic.Int64
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
	w.wg.Add(1)
	var restartCount int
	go func() {
		defer w.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "worker goroutine panicked, restarting", "panic", r)
				sentry.CurrentHub().RecoverWithContext(ctx, r)
				restartCount++
				backoff := time.Duration(restartCount) * time.Second
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
				time.Sleep(backoff)
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
			sentry.CurrentHub().RecoverWithContext(ctx, r)
		}
	}()
	job.Execute(ctx)
}

func (w *Worker) Dispatch(job Job) {
	select {
	case w.jobs <- job:
		slog.Debug("job dispatched", "type", job.Type())
	default:
		dropped := w.droppedTotal.Add(1)
		fields := append([]any{"type", job.Type(), "dropped_total", dropped}, job.LogFields()...)
		slog.Warn("worker queue full, dropping job", fields...)
	}
}

func (w *Worker) Drain(timeout time.Duration) {
	done := make(chan struct{})
	go func() { w.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(timeout):
		slog.Warn("worker drain timed out")
	}
}

func (w *Worker) DroppedTotal() int64 {
	return w.droppedTotal.Load()
}
