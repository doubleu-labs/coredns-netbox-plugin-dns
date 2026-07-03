package poller

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/prometheus/client_golang/prometheus"
)

var defaultViewPollerInterval = 30 * time.Second

type ViewPoller struct {
	cancel   context.CancelFunc
	client   *api.Client
	interval time.Duration
	mu       sync.Mutex
	running  bool
	wg       sync.WaitGroup

	configuredViews core.Views
	runtimeViews    core.Views

	canResolve          bool
	lastError           error
	missingIncludeSlice []string
	missingExcludeSlice []string
	lastSuccessfulPoll  time.Time

	pollerCyclesMetric   *prometheus.CounterVec
	pollerDurationMetric prometheus.Histogram
	pollerErrorsMetric   *prometheus.CounterVec
}

func NewViewPoller(c *api.Client, v *core.Views, i time.Duration) (
	*ViewPoller,
	error,
) {
	if c == nil {
		return nil, errors.New("view poller client is required")
	}
	if v == nil {
		return nil, errors.New("view poller configured views are required")
	}
	interval := i
	if i <= 0 {
		interval = defaultViewPollerInterval
	}
	vp := &ViewPoller{
		client:               c,
		configuredViews:      cloneViews(*v),
		interval:             interval,
		runtimeViews:         cloneViews(*v),
		canResolve:           true,
		pollerCyclesMetric:   newViewPollerCyclesTotalMetric(),
		pollerDurationMetric: newViewPollerPollDurationMetric(),
		pollerErrorsMetric:   newViewPollerErrorsMetric(),
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

		if err := vp.poll(ctx, time.Now()); err != nil && !errors.Is(
			err,
			context.Canceled,
		) {
			vp.pollerErrorsMetric.WithLabelValues("start").Inc()
		}

		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				if err := vp.poll(ctx, t); err != nil && !errors.Is(
					err,
					context.Canceled,
				) {
					vp.pollerErrorsMetric.WithLabelValues("poll").Inc()
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

func (vp *ViewPoller) CanResolve() bool {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	return vp.canResolve
}

func (vp *ViewPoller) Views() core.Views {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	return cloneViews(vp.runtimeViews)
}

func (vp *ViewPoller) SetCanResolveForTest(canResolve bool) {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	vp.canResolve = canResolve
}

func (vp *ViewPoller) lastErr() error {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	return vp.lastError
}

func (vp *ViewPoller) missingInclude() []string {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	return slices.Clone(vp.missingIncludeSlice)
}

func (vp *ViewPoller) missingExclude() []string {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	return slices.Clone(vp.missingExcludeSlice)
}

func (vp *ViewPoller) poll(ctx context.Context, _ time.Time) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		break
	}

	rq := &api.ViewsQuery{
		Brief: true,
	}
	start := time.Now()
	defer func() {
		vp.pollerDurationMetric.Observe(time.Since(start).Seconds())
	}()
	views, err := rq.GetViews(ctx, vp.client)
	if err != nil {
		vp.setLastErr(err)
		vp.pollerCyclesMetric.WithLabelValues("error").Inc()
		return err
	}
	vp.pollerCyclesMetric.WithLabelValues("success").Inc()

	serverViews := make(map[string]struct{}, len(views))
	for view := range slices.Values(views) {
		serverViews[view.Name] = struct{}{}
	}

	runtime, mInclude, mExclude, canResolve := vp.validate(serverViews)

	vp.mu.Lock()
	vp.runtimeViews = runtime
	vp.missingIncludeSlice = mInclude
	vp.missingExcludeSlice = mExclude
	vp.canResolve = canResolve
	vp.lastSuccessfulPoll = time.Now()

	if !canResolve {
		vp.lastError = fmt.Errorf(
			"none of the configured include views exist in netbox: %s",
			strings.Join(vp.configuredViews.Include, ", "),
		)
	} else {
		vp.lastError = nil
	}

	err = vp.lastError
	vp.mu.Unlock()

	return err
}

func (vp *ViewPoller) validate(serverViews map[string]struct{}) (
	core.Views,
	[]string,
	[]string,
	bool,
) {
	vp.mu.Lock()
	configured := cloneViews(vp.configuredViews)
	vp.mu.Unlock()

	runtime := core.Views{
		Include: existingViews(configured.Include, serverViews),
		Exclude: existingViews(configured.Exclude, serverViews),
	}

	missingInclude := missingViews(configured.Include, serverViews)
	missingExclude := missingViews(configured.Exclude, serverViews)

	canResolve := true
	if len(configured.Include) > 0 && len(runtime.Include) == 0 {
		canResolve = false
	}

	return runtime, missingInclude, missingExclude, canResolve
}

func (vp *ViewPoller) setLastErr(err error) {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	vp.lastError = err
}

func existingViews(names []string, serverViews map[string]struct{}) []string {
	out := make([]string, 0, len(names))
	for name := range slices.Values(names) {
		if _, ok := serverViews[name]; ok {
			out = append(out, name)
		}
	}
	return out
}

func missingViews(names []string, serverViews map[string]struct{}) []string {
	out := make([]string, 0, len(names))
	for name := range slices.Values(names) {
		if _, ok := serverViews[name]; !ok {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

func cloneViews(v core.Views) core.Views {
	return core.Views{
		Include: slices.Clone(v.Include),
		Exclude: slices.Clone(v.Exclude),
	}
}
