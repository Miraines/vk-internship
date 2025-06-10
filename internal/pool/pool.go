package pool

import (
	"context"
	"fmt"
	"sync"
)

type WorkerPool struct {
	mu      sync.Mutex
	workers map[int]*worker
	nextID  int
	jobs    chan string
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

type worker struct {
	id   int
	jobs <-chan string
	quit chan struct{}
	wg   *sync.WaitGroup
}

func New(queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workers: make(map[int]*worker),
		jobs:    make(chan string, queueSize),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (p *WorkerPool) Submit(s string) {
	select {
	case p.jobs <- s:
	case <-p.ctx.Done():
	}
}

func (p *WorkerPool) AddWorker() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	id := p.nextID
	p.nextID++

	w := &worker{
		id:   id,
		jobs: p.jobs,
		quit: make(chan struct{}),
		wg:   &p.wg,
	}
	p.workers[id] = w
	p.wg.Add(1)
	go w.run()

	return id
}

func (p *WorkerPool) RemoveWorker(id int) {
	p.mu.Lock()
	w, ok := p.workers[id]
	if ok {
		delete(p.workers, id)
	}
	p.mu.Unlock()

	if ok {
		close(w.quit)
	}
}

func (p *WorkerPool) Shutdown() {
	p.cancel()

	p.mu.Lock()
	for id, w := range p.workers {
		delete(p.workers, id)
		close(w.quit)
	}
	p.mu.Unlock()

	close(p.jobs)

	p.wg.Wait()
}

func (w *worker) run() {
	defer w.wg.Done()
	for {
		select {
		case s, ok := <-w.jobs:
			if !ok {
				return
			}
			fmt.Printf("[worker %d] %s\n", w.id, s)
		case <-w.quit:
			return
		}
	}
}
