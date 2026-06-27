package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pollDurationOnce   sync.Once
	pollDurationMetric prometheus.Histogram
)

type PollDuration struct {
	metric prometheus.Histogram
}

func NewPollDuration(subsystem string) PollDuration {
	o := prometheus.HistogramOpts{
		Namespace: plugin.Namespace,
		Subsystem: subsystem,
		Name:      "poll_duration_seconds",
		Help:      "histogram of poll-cycle execution latency",
		Buckets:   prometheus.ExponentialBuckets(0.01, 2, 14),
	}
	pollDurationOnce.Do(
		func() {
			pollDurationMetric = promauto.NewHistogram(o)
		},
	)
	return PollDuration{
		metric: pollDurationMetric,
	}
}

func (p *PollDuration) Observe(d float64) {
	p.metric.Observe(d)
}
