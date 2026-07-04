package poller

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type pollFunc func(context.Context) error

type Poller struct {
	cancel      context.CancelFunc
	ctx         context.Context
	ErrorMetric *prometheus.CounterVec
	Interval    time.Duration
	Mu          sync.Mutex
	PollFunc    pollFunc
	running     bool
	wg          sync.WaitGroup
}

func (p *Poller) Start() {
	p.Mu.Lock()
	if p.running {
		p.Mu.Unlock()
		return
	}

	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.running = true
	p.wg.Add(1)
	p.Mu.Unlock()

	go p.runLoop()
}

func (p *Poller) Stop() {
	p.Mu.Lock()
	if !p.running {
		p.Mu.Unlock()
		return
	}

	cancel := p.cancel
	p.cancel = nil
	p.running = false
	p.Mu.Unlock()

	cancel()
	p.wg.Wait()
}

func (p *Poller) runLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	p.runPoll("start")

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.runPoll("poll")
		}
	}
}

func (p *Poller) runPoll(whenLabel string) {
	if err := p.PollFunc(p.ctx); err != nil && !errors.Is(
		err,
		context.Canceled,
	) {
		p.ErrorMetric.WithLabelValues(whenLabel).Inc()
	}
}
