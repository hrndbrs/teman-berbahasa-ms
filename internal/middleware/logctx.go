package middleware

import (
	"context"
	"sync"
)

type logRecord struct {
	mu     sync.Mutex
	fields []any
}

func (l *logRecord) add(kvs ...any) {
	l.mu.Lock()
	l.fields = append(l.fields, kvs...)
	l.mu.Unlock()
}

func (l *logRecord) snapshot() []any {
	l.mu.Lock()
	out := make([]any, len(l.fields))
	copy(out, l.fields)
	l.mu.Unlock()
	return out
}

type contextKeyLog struct{}

func logRecordFromCtx(ctx context.Context) *logRecord {
	rec, _ := ctx.Value(contextKeyLog{}).(*logRecord)
	return rec
}

// AddLogFields enriches the current request's log line with extra key-value pairs.
// Safe to call from any middleware or handler running in the request's goroutine.
func AddLogFields(ctx context.Context, kvs ...any) {
	if rec := logRecordFromCtx(ctx); rec != nil {
		rec.add(kvs...)
	}
}
