package pool

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTaskProcessing(t *testing.T) {
	var processed int32
	p := New(
		10,
		WithInitialWorkers(2),
		WithUserHandler(func(ctx context.Context, job string) {
			atomic.AddInt32(&processed, 1)
		}),
	)
	defer p.Shutdown()

	for i := 0; i < 5; i++ {
		p.Submit("job")
	}

	time.Sleep(200 * time.Millisecond)
	if atomic.LoadInt32(&processed) != 5 {
		t.Fatalf("expected 5 processed, got %d", processed)
	}
}

func TestAddRemoveWorkers(t *testing.T) {
	p := New(0)

	if got := p.WorkerCount(); got != 1 {
		t.Fatalf("expected 1 worker start, got %d", got)
	}
	id := p.AddWorker()
	if got := p.WorkerCount(); got != 2 {
		t.Fatalf("AddWorker: expected 2, got %d", got)
	}
	p.RemoveWorker(id)
	time.Sleep(20 * time.Millisecond)
	if got := p.WorkerCount(); got != 1 {
		t.Fatalf("RemoveWorker: expected 1, got %d", got)
	}
	p.Shutdown()
}

func TestAutoScaling(t *testing.T) {
	p := New(
		100,
		WithInitialWorkers(1),
		WithAutoScaler(AutoScaleConfig{
			Min:          1,
			Max:          4,
			UpThreshold:  2,
			ObserveEvery: 50 * time.Millisecond,
		}),
		WithUserHandler(func(ctx context.Context, job string) {
			time.Sleep(20 * time.Millisecond)
		}),
	)
	defer p.Shutdown()

	for i := 0; i < 10; i++ {
		p.Submit("load")
	}

	grew := false
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if p.WorkerCount() > 1 {
			grew = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !grew {
		t.Fatalf("autoscale up never happened, workers=%d", p.WorkerCount())
	}

	time.Sleep(600 * time.Millisecond)
	if workers := p.WorkerCount(); workers < 1 || workers > 4 {
		t.Fatalf("autoscale produced invalid count: %d", workers)
	}
}

func TestMiddlewareAndRecover(t *testing.T) {
	var called, recovered int32

	logMw := Middleware(func(next Handler) Handler {
		return func(ctx context.Context, job string) {
			atomic.AddInt32(&called, 1)
			next(ctx, job)
		}
	})
	recMw := RecoverMiddleware(nil)

	p := New(
		5,
		WithMiddleware(logMw, recMw),
		WithUserHandler(func(ctx context.Context, job string) {
			panic("boom")
		}),
	)
	defer p.Shutdown()

	p.Submit("one")
	p.Submit("two")

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&called) != 2 {
		t.Fatalf("middleware not executed expected=2 got=%d", called)
	}
	atomic.StoreInt32(&recovered, 1)
}

func TestRateLimit(t *testing.T) {
	const rps = 10
	const jobs = 20

	var ts []time.Time
	var mu sync.Mutex

	p := New(
		10,
		WithRateLimit(rps),
		WithUserHandler(func(ctx context.Context, job string) {
			mu.Lock()
			ts = append(ts, time.Now())
			mu.Unlock()
		}),
	)
	defer p.Shutdown()

	for i := 0; i < jobs; i++ {
		p.Submit("rl-job")
	}
	time.Sleep(3 * time.Second)

	mu.Lock()
	defer mu.Unlock()
	if len(ts) != jobs {
		t.Fatalf("expected %d jobs processed, got %d", jobs, len(ts))
	}
	for i := 1; i < len(ts); i++ {
		if delta := ts[i].Sub(ts[i-1]); delta < 80*time.Millisecond {
			t.Fatalf("rate-limit broke: delta=%s between job %d/%d", delta, i, jobs)
		}
	}
}
