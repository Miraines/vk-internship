package pool

import (
	"context"
	"sync"
)

type Handler func(ctx context.Context, job string)

type WorkerPool struct {
	jobs chan string

	opts *options

	mu      sync.Mutex
	workers map[int]*worker
	nextID  int

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type worker struct {
	id       int
	parent   *WorkerPool
	quitChan chan struct{}
}

func New(queueSize int, setters ...Option) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	opt := defaultOptions()
	for _, s := range setters {
		s(opt)
	}

	p := &WorkerPool{
		jobs:    make(chan string, queueSize),
		opts:    opt,
		workers: make(map[int]*worker),
		ctx:     ctx,
		cancel:  cancel,
	}

	for i := 0; i < opt.initialWorkers; i++ {
		p.addWorkerUnlocked()
	}
	if opt.autoScale.enabled {
		go p.autoScalerLoop()
	}

	return p
}

func (p *WorkerPool) SubmitContext(ctx context.Context, s string) {
	select {
	case p.jobs <- s:
	case <-ctx.Done():
	}
}

func (p *WorkerPool) Submit(s string) { p.SubmitContext(context.Background(), s) }

func (p *WorkerPool) AddWorker() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.addWorkerUnlocked()
}

func (p *WorkerPool) RemoveWorker(id int) {
	p.mu.Lock()
	w, ok := p.workers[id]
	if ok {
		delete(p.workers, id)
	}
	p.mu.Unlock()
	if ok {
		close(w.quitChan)
	}
}

func (p *WorkerPool) Shutdown() {
	p.cancel()
	close(p.jobs)

	p.mu.Lock()
	for _, w := range p.workers {
		close(w.quitChan)
	}
	p.mu.Unlock()

	p.wg.Wait()
}

func (p *WorkerPool) WorkerCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.workers)
}

func (p *WorkerPool) addWorkerUnlocked() int {
	id := p.nextID
	p.nextID++

	w := &worker{
		id:       id,
		parent:   p,
		quitChan: make(chan struct{}),
	}
	p.workers[id] = w
	p.wg.Add(1)
	go w.loop()
	p.opts.logger.Printf("[pool] worker %d started (total=%d)", id, len(p.workers))
	return id
}

func (w *worker) loop() {
	defer func() {
		w.parent.wg.Done()
		w.parent.opts.logger.Printf("[pool] worker %d stopped", w.id)
	}()

	handler := w.parent.opts.composeMiddleware(w.parent.opts.userHandler)

	for {
		select {
		case job, ok := <-w.parent.jobs:
			if !ok {
				return
			}
			handler(w.parent.ctx, job)

		case <-w.quitChan:
			return
		}
	}
}
