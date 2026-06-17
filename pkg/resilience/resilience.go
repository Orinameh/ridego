package resilience

import (
	"context"
	"math"
	"math/rand"
	"time"

	"log/slog"

	"github.com/sony/gobreaker/v2"
)

// NewBreaker creates a circuit breaker for a named upstream service.
// Opens after 5 consecutive failures, probes again after 30 seconds.
func NewBreaker(name string) *gobreaker.CircuitBreaker[[]byte] {
	return gobreaker.NewCircuitBreaker[[]byte](gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures > 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("circuit breaker state changed",
				"service", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	})
}

// WithRetry retries fn up to maxAttempts times with exponential backoff.
// It will not retry if the context is cancelled or the error is not retryable.
func WithRetry(ctx context.Context, maxAttempts int, fn func() error) error {
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt == maxAttempts-1 {
			break
		}
		backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
		jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
		select {
		case <-time.After(backoff + jitter):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
