package mainserverclient

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	maxRetries  = 5
	maxQueueLen = 50
)

var retryBackoff = []time.Duration{
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
	30 * time.Second,
	60 * time.Second,
}

type queuedPost struct {
	path     string
	body     any
	attempts int
	nextTry  time.Time
}

// RetryQueue holds failed API posts for background retry.
type RetryQueue struct {
	mu          sync.Mutex
	items       []queuedPost
	postFn      func(ctx context.Context, path string, body any) error
	authBlocked bool
}

func newRetryQueue(postFn func(ctx context.Context, path string, body any) error) *RetryQueue {
	return &RetryQueue{postFn: postFn}
}

func (q *RetryQueue) SetAuthBlocked(blocked bool) {
	q.mu.Lock()
	q.authBlocked = blocked
	q.mu.Unlock()
}

func (q *RetryQueue) AuthBlocked() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.authBlocked
}

// Enqueue adds a failed post for later retry.
func (q *RetryQueue) Enqueue(path string, body any) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.authBlocked {
		return
	}
	item := queuedPost{
		path:    path,
		body:    body,
		nextTry: time.Now().Add(retryBackoff[0]),
	}
	if len(q.items) >= maxQueueLen {
		log.Warn().Str("path", path).Msg("main server retry queue full; dropping oldest item")
		q.items = q.items[1:]
	}
	q.items = append(q.items, item)
}

// Flush attempts all due items; returns count still pending.
func (q *RetryQueue) Flush(ctx context.Context) int {
	q.mu.Lock()
	if q.authBlocked || len(q.items) == 0 {
		n := len(q.items)
		q.mu.Unlock()
		return n
	}
	pending := append([]queuedPost(nil), q.items...)
	q.mu.Unlock()

	var remaining []queuedPost
	for _, item := range pending {
		if ctx.Err() != nil {
			remaining = append(remaining, item)
			continue
		}
		if time.Now().Before(item.nextTry) {
			remaining = append(remaining, item)
			continue
		}
		err := q.postFn(ctx, item.path, item.body)
		if err == nil {
			continue
		}
		if IsAuthError(err) {
			q.SetAuthBlocked(true)
			log.Warn().Msg("main server auth failed; stopping API retries until token fixed")
			return len(pending)
		}
		item.attempts++
		if item.attempts >= maxRetries || !IsRetryable(err) {
			log.Warn().Err(err).Str("path", item.path).Msg("main server post dropped after retries")
			continue
		}
		idx := item.attempts
		if idx >= len(retryBackoff) {
			idx = len(retryBackoff) - 1
		}
		item.nextTry = time.Now().Add(retryBackoff[idx])
		remaining = append(remaining, item)
	}

	q.mu.Lock()
	q.items = remaining
	n := len(q.items)
	q.mu.Unlock()
	return n
}

// StartFlusher runs periodic flush until ctx is cancelled.
func (q *RetryQueue) StartFlusher(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = q.Flush(ctx)
			}
		}
	}()
}
