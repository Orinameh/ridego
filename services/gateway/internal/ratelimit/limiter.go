package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter is a sliding-window rate limiter backed by Redis.
// Key: ridego:rl:{userID} (falls back to IP for unauthenticated requests)
type Limiter struct {
	rdb    *redis.Client
	max    int64
	window time.Duration
}

func New(rdb *redis.Client, max int64, window time.Duration) *Limiter {
	return &Limiter{rdb: rdb, max: max, window: window}
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := l.keyFor(r)
		allowed, remaining, err := l.check(r.Context(), key)
		if err != nil {
			// Redis failure → fail open (allow request)
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", l.max))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(l.window.Seconds())))
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) check(ctx context.Context, key string) (allowed bool, remaining int64, err error) {
	now := time.Now()
	start := now.Add(-l.window)

	pipe := l.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", start.UnixNano()))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: now.UnixNano()})
	pipe.Expire(ctx, key, l.window)
	if _, err = pipe.Exec(ctx); err != nil {
		return false, 0, err
	}

	count := countCmd.Val()
	remaining = l.max - count - 1
	if remaining < 0 {
		remaining = 0
	}
	return count < l.max, remaining, nil
}

func (l *Limiter) keyFor(r *http.Request) string {
	if uid := r.Header.Get("X-User-ID"); uid != "" {
		return "ridego:rl:" + uid
	}
	return "ridego:rl:ip:" + r.RemoteAddr
}
