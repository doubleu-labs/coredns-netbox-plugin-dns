package poller

import (
	"context"
	"sync"
	"time"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
)

var defaultViewPollerInterval = 30 * time.Second

type ViewPoller struct {
	cancel   context.CancelFunc
	client   *api.Client
	interval time.Duration
	mu       sync.Mutex
	running  bool
	wg       sync.WaitGroup
}

func NewViewPoller(i time.Duration) (*ViewPoller, error) {
	interval := i
	if i <= 0 {
		interval = defaultViewPollerInterval
	}
	vp := &ViewPoller{
		interval: interval,
	}
	return vp, nil
}

func (vp *ViewPoller) Start() {
	vp.mu.Lock()
	if vp.running {
		vp.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	vp.cancel = cancel
	vp.running = true
	vp.wg.Add(1)
	vp.mu.Unlock()

	go func() {
		defer vp.wg.Done()

		ticker := time.NewTicker(vp.interval)
		defer ticker.Stop()

		if err := vp.poll(ctx, time.Now()); err != nil {
			// TODO: log/record metric
		}

		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				if err := vp.poll(ctx, t); err != nil {
					// TODO: log/record metric
				}
			}
		}

	}()
}

func (vp *ViewPoller) Stop() {
	vp.mu.Lock()
	if !vp.running {
		vp.mu.Unlock()
		return
	}

	cancel := vp.cancel
	vp.cancel = nil
	vp.running = false
	vp.mu.Unlock()

	cancel()
	vp.wg.Wait()
}

func (vp *ViewPoller) poll(ctx context.Context, _ time.Time) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		break
	}

	// TODO: implement
	return nil
}
