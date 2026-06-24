package metrics

import (
	"net/http"
	"strings"
	"time"
)

const netboxEndpointLabelPathPrefix = "/api/plugins/netbox-dns/"

type InstrumentedTransport struct {
	base    http.RoundTripper
	metrics *Metrics
}

func NewInstrumentedTransport(
	rt http.RoundTripper,
	m *Metrics,
) *InstrumentedTransport {
	return &InstrumentedTransport{
		base:    rt,
		metrics: m,
	}
}

func (i *InstrumentedTransport) RoundTrip(r *http.Request) (
	*http.Response,
	error,
) {
	e := netboxEndpointLabel(r.URL.Path)
	s := time.Now()
	resp, err := i.base.RoundTrip(r)
	i.metrics.NetboxRequestDuration.WithLabelValues(e).
		Observe(time.Since(s).Seconds())
	if err != nil {
		i.metrics.NetboxRequestsTotal.WithLabelValues(e, "error").Inc()
		return resp, err
	}
	i.metrics.NetboxRequestsTotal.WithLabelValues(
		e,
		http.StatusText(resp.StatusCode),
	).Inc()
	return resp, nil
}

func netboxEndpointLabel(p string) string {
	if !strings.HasPrefix(p, netboxEndpointLabelPathPrefix) {
		return "other"
	}
	r := strings.TrimPrefix(p, netboxEndpointLabelPathPrefix)
	if i := strings.IndexByte(r, '/'); i >= 0 {
		r = r[:i]
	}
	if r == "" {
		return "other"
	}
	return r
}
