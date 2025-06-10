package pool

import (
	"context"
	"log"
	"time"
)

type Middleware func(next Handler) Handler

func LoggingMiddleware(l *log.Logger) Middleware {
	if l == nil {
		l = log.Default()
	}
	return func(next Handler) Handler {
		return func(ctx context.Context, job string) {
			start := time.Now()
			l.Printf("[mw] start job=%q", job)
			next(ctx, job)
			l.Printf("[mw] done  job=%q latency=%s", job, time.Since(start))
		}
	}
}

func RecoverMiddleware(l *log.Logger) Middleware {
	if l == nil {
		l = log.Default()
	}
	return func(next Handler) Handler {
		return func(ctx context.Context, job string) {
			defer func() {
				if r := recover(); r != nil {
					l.Printf("[mw] panic recovered: %v", r)
				}
			}()
			next(ctx, job)
		}
	}
}

func RetryMiddleware(maxRetries int) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, job string) {
			delay := time.Millisecond * 50
			for i := 0; i <= maxRetries; i++ {
				done := make(chan struct{})
				go func() {
					defer close(done)
					next(ctx, job)
				}()
				select {
				case <-done:
					return
				case <-time.After(delay):
					delay *= 2
				}
			}
		}
	}
}

func RateLimitMiddleware(rps int) Middleware {
	interval := time.Second / time.Duration(rps)
	tokens := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case tokens <- struct{}{}:
			default:
			}
		}
	}()
	return func(next Handler) Handler {
		return func(ctx context.Context, job string) {
			<-tokens
			next(ctx, job)
		}
	}
}
