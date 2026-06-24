package metrics

import "time"

func (m *Metrics) MeasureRequestDuration(zone string, t time.Time) {
	m.RequestDuration.WithLabelValues(zone).Observe(
		time.Since(t).Seconds(),
	)
}
