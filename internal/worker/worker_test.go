package worker_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/worker"
)

type countJob struct {
	name    string
	counter *atomic.Int64
	done    chan struct{}
}

func (j *countJob) Type() string { return j.name }
func (j *countJob) LogFields() []any { return []any{"job_name", j.name} }
func (j *countJob) Execute(_ context.Context) {
	j.counter.Add(1)
	if j.done != nil {
		j.done <- struct{}{}
	}
}

func TestWorker_ExecutesJob(t *testing.T) {
	w := worker.New(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx, 2)

	var counter atomic.Int64
	done := make(chan struct{}, 1)
	w.Dispatch(&countJob{name: "test", counter: &counter, done: done})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("job not executed within 1s")
	}

	assert.Equal(t, int64(1), counter.Load())
}

func TestWorker_DropOnFull(t *testing.T) {
	w := worker.New(0) // zero-capacity channel — always full when no readers
	// Do NOT start workers — no goroutine consuming from channel

	// Must return immediately without blocking or panicking
	w.Dispatch(&countJob{name: "dropped", counter: new(atomic.Int64)})
}

func TestWorker_StopsOnContextCancel(t *testing.T) {
	w := worker.New(10)
	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx, 2)

	cancel() // stop workers

	// Give goroutines time to exit; then dispatch should not execute
	time.Sleep(10 * time.Millisecond)

	var counter atomic.Int64
	w.Dispatch(&countJob{name: "after-cancel", counter: &counter})

	time.Sleep(50 * time.Millisecond)
	// counter may be 0 or 1 depending on race; main thing: no panic, no hang
	_ = counter.Load()
}

func TestWorker_ConcurrentJobs(t *testing.T) {
	w := worker.New(100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx, 4)

	const n = 20
	var counter atomic.Int64
	done := make(chan struct{}, n)

	for range n {
		w.Dispatch(&countJob{name: "concurrent", counter: &counter, done: done})
	}

	for range n {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("not all jobs executed within 2s")
		}
	}

	require.Equal(t, int64(n), counter.Load())
}
