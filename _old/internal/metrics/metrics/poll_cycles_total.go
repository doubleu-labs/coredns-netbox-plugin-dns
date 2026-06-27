package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	pollCyclesTotalOnce   sync.Once
	pollCyclesTotalMetric *prometheus.CounterVec
)

type PollCyclesTotal struct {
	metric *prometheus.CounterVec
}

func NewPollCyclesTotal(subsystem string) *PollCyclesTotal {
	o := prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: subsystem,
		Name:      "poll_cycles_total",
		Help:      "counter of full poll-cycle executions",
	}
	pollCyclesTotalOnce.Do(
		func() {
			pollCyclesTotalMetric = prometheus.NewCounterVec(
				o,
				[]string{"result"},
			)
		},
	)
	return &PollCyclesTotal{
		metric: pollCyclesTotalMetric,
	}
}

func (p *PollCyclesTotal) Inc(result string) {
	p.metric.WithLabelValues(result).Inc()
}
