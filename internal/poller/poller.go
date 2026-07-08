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

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.ctx = ctx
	p.running = true
	p.wg.Add(1)
	p.Mu.Unlock()

	go p.runLoop(ctx)
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

func (p *Poller) runLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	p.runPoll(ctx, "start")

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.runPoll(ctx, "poll")
		}
	}
}

func (p *Poller) runPoll(ctx context.Context, whenLabel string) {
	if err := p.PollFunc(ctx); err != nil &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded) {
		p.ErrorMetric.WithLabelValues(whenLabel).Inc()
	}
}
