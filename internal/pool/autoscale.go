package pool

import "time"

type AutoScaleConfig struct {
	Min          int           // минимум воркеров
	Max          int           // максимум воркеров
	UpThreshold  int           // очередь > threshold → scale-up
	ObserveEvery time.Duration // частота проверки
}

type autoScaleConfig struct {
	enabled bool
	AutoScaleConfig
}

func defaultAutoScale() autoScaleConfig {
	return autoScaleConfig{
		enabled: false,
		AutoScaleConfig: AutoScaleConfig{
			Min:          1,
			Max:          8,
			UpThreshold:  10,
			ObserveEvery: time.Second,
		},
	}
}

func (p *WorkerPool) autoScalerLoop() {
	cfg := p.opts.autoScale
	ticker := time.NewTicker(cfg.ObserveEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			q := len(p.jobs)

			p.mu.Lock()
			workers := len(p.workers)

			scaleUp := q > cfg.UpThreshold && workers < cfg.Max
			scaleDown := q == 0 && workers > cfg.Min

			switch {
			case scaleUp:
				add := min(cfg.Max-workers, workers)
				for i := 0; i < add; i++ {
					p.addWorkerUnlocked()
				}
				p.opts.logger.Printf("[autoscale] scale UP → %d workers (queue=%d)", len(p.workers), q)

			case scaleDown:
				remove := workers / 2
				ids := make([]int, 0, remove)
				for id := range p.workers {
					if len(ids) == remove {
						break
					}
					ids = append(ids, id)
				}
				for _, id := range ids {
					close(p.workers[id].quitChan)
					delete(p.workers, id)
				}
				p.opts.logger.Printf("[autoscale] scale DOWN → %d workers", len(p.workers))
			}
			p.mu.Unlock()

		case <-p.ctx.Done():
			return
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
