package mainserverclient

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryQueueAuthStopsImmediately(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantBlock bool
	}{
		{name: "auth", err: &APIError{StatusCode: 401, Message: "invalid token"}, wantBlock: true},
		{name: "retryable", err: &APIError{StatusCode: 503, Message: "down"}, wantBlock: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := newRetryQueue(func(ctx context.Context, path string, body any) error {
				return tc.err
			})
			q.mu.Lock()
			q.items = []queuedPost{{path: "/api/collector/heartbeat", body: map[string]string{"x": "y"}, nextTry: time.Now().Add(-time.Second)}}
			q.mu.Unlock()
			q.Flush(context.Background())
			if q.AuthBlocked() != tc.wantBlock {
				t.Fatalf("AuthBlocked=%v want %v", q.AuthBlocked(), tc.wantBlock)
			}
		})
	}
}

func TestRetryQueueEventuallySucceeds(t *testing.T) {
	var calls atomic.Int32
	q := newRetryQueue(func(ctx context.Context, path string, body any) error {
		if calls.Add(1) < 2 {
			return errors.New("network down")
		}
		return nil
	})
	q.mu.Lock()
	q.items = []queuedPost{{path: "/api/x", body: map[string]int{"a": 1}, nextTry: time.Now().Add(-time.Second)}}
	q.mu.Unlock()
	q.Flush(context.Background())
	q.mu.Lock()
	q.items[0].nextTry = time.Now().Add(-time.Second)
	q.mu.Unlock()
	q.Flush(context.Background())
	q.mu.Lock()
	n := len(q.items)
	q.mu.Unlock()
	if n != 0 {
		t.Fatalf("expected empty queue, got %d items", n)
	}
}

func TestRetryQueueCapDropsOldest(t *testing.T) {
	q := newRetryQueue(func(ctx context.Context, path string, body any) error {
		return errors.New("fail")
	})
	for i := 0; i < maxQueueLen+2; i++ {
		q.Enqueue("/api/x", i)
	}
	q.mu.Lock()
	n := len(q.items)
	q.mu.Unlock()
	if n != maxQueueLen {
		t.Fatalf("queue len=%d want %d", n, maxQueueLen)
	}
}
