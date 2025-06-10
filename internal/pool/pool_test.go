package pool

import (
	"testing"
	"time"
)

func TestAddAndRemove(t *testing.T) {
	p := New(0)
	id := p.AddWorker()
	if len(p.workers) != 1 {
		t.Fatalf("expected 1 worker, got %d", len(p.workers))
	}
	p.RemoveWorker(id)
	time.Sleep(10 * time.Millisecond)
	if len(p.workers) != 0 {
		t.Fatalf("expected 0 workers, got %d", len(p.workers))
	}
	p.Shutdown()
}
