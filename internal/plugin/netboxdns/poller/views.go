package poller

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/poller"
	"github.com/prometheus/client_golang/prometheus"
)

var defaultViewPollerInterval = 30 * time.Second

type ViewPoller struct {
	*poller.Poller
	client *api.Client

	configuredViews core.Views
	runtimeViews    core.Views

	canResolve          bool
	lastError           error
	missingIncludeSlice []string
	missingExcludeSlice []string
	lastSuccessfulPoll  time.Time

	pollerCyclesMetric   *prometheus.CounterVec
	pollerDurationMetric prometheus.Histogram
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
		Poller: &poller.Poller{
			ErrorMetric: newViewPollerErrorsMetric(),
			Interval:    interval,
		},
		client:               c,
		configuredViews:      cloneViews(*v),
		runtimeViews:         cloneViews(*v),
		canResolve:           true,
		pollerCyclesMetric:   newViewPollerCyclesTotalMetric(),
		pollerDurationMetric: newViewPollerPollDurationMetric(),
	}
	vp.Poller.PollFunc = vp.poll
	return vp, nil
}

func (vp *ViewPoller) CanResolve() bool {
	vp.Poller.Mu.Lock()
	defer vp.Poller.Mu.Unlock()
	return vp.canResolve
}

func (vp *ViewPoller) Views() core.Views {
	vp.Poller.Mu.Lock()
	defer vp.Poller.Mu.Unlock()
	return cloneViews(vp.runtimeViews)
}

func (vp *ViewPoller) SetCanResolveForTest(canResolve bool) {
	vp.Poller.Mu.Lock()
	defer vp.Poller.Mu.Unlock()
	vp.canResolve = canResolve
}

func (vp *ViewPoller) lastErr() error {
	vp.Poller.Mu.Lock()
	defer vp.Poller.Mu.Unlock()
	return vp.lastError
}

func (vp *ViewPoller) missingInclude() []string {
	vp.Poller.Mu.Lock()
	defer vp.Poller.Mu.Unlock()
	return slices.Clone(vp.missingIncludeSlice)
}

func (vp *ViewPoller) missingExclude() []string {
	vp.Poller.Mu.Lock()
	defer vp.Poller.Mu.Unlock()
	return slices.Clone(vp.missingExcludeSlice)
}

func (vp *ViewPoller) poll(ctx context.Context) error {
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

	vp.Poller.Mu.Lock()
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
	vp.Poller.Mu.Unlock()

	return err
}

func (vp *ViewPoller) validate(serverViews map[string]struct{}) (
	core.Views,
	[]string,
	[]string,
	bool,
) {
	vp.Poller.Mu.Lock()
	configured := cloneViews(vp.configuredViews)
	vp.Poller.Mu.Unlock()

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
	vp.Poller.Mu.Lock()
	defer vp.Poller.Mu.Unlock()
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
