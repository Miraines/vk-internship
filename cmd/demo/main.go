package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"vk-internship/internal/pool"
)

func main() {
	p := pool.New(
		100,
		pool.WithInitialWorkers(2),
		pool.WithLogger(log.Default()),
		pool.WithUserHandler(func(ctx context.Context, job string) {
			// эмулируем работу
			fmt.Printf("[handler] %s (goroutine=%d)\n", job, time.Now().UnixNano()%1e4)
			time.Sleep(100 * time.Millisecond)
		}),
		pool.WithMiddleware(
			pool.LoggingMiddleware(nil),
			pool.RecoverMiddleware(nil),
			pool.RetryMiddleware(2),
		),
		pool.WithAutoScaler(pool.AutoScaleConfig{
			Min:          2,
			Max:          16,
			UpThreshold:  5,
			ObserveEvery: time.Second,
		}),
		pool.WithRateLimit(50),
	)
	defer p.Shutdown()

	go func() {
		for i := 1; i <= 30; i++ {
			p.Submit(fmt.Sprintf("task-%02d", i))
		}
	}()

	time.Sleep(4 * time.Second)
}
