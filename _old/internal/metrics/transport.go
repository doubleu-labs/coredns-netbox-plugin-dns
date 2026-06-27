package metrics

import (
	"net/http"
	"strings"
	"time"
)

const (
	netboxEndpointLabelPathPrefix = "/api/plugins/netbox-dns/"
	fallbackEndpointLabel         = "other"
)

type InstrumentedTransport struct {
	http.RoundTripper
	metrics *Metrics
}

func NewInstrumentedTransport(
	rt http.RoundTripper,
	m *Metrics,
) *InstrumentedTransport {
	return &InstrumentedTransport{
		RoundTripper: rt,
		metrics:      m,
	}
}

func (i *InstrumentedTransport) RoundTrip(r *http.Request) (
	*http.Response,
	error,
) {
	e := netboxEndpointLabel(r.URL.Path)
	s := time.Now()
	resp, err := i.RoundTrip(r)
	i.metrics.NetboxRequestDuration.Observe(e, time.Since(s).Seconds())
	if err != nil {
		i.metrics.NetboxRequestsTotal.Inc(e, "error")
		return resp, err
	}
	i.metrics.NetboxRequestsTotal.Inc(e, http.StatusText(resp.StatusCode))
	return resp, nil
}

func netboxEndpointLabel(p string) string {
	if !strings.HasPrefix(p, netboxEndpointLabelPathPrefix) {
		return fallbackEndpointLabel
	}
	r := strings.TrimPrefix(p, netboxEndpointLabelPathPrefix)
	if i := strings.IndexByte(r, '/'); i >= 0 {
		r = r[:i]
	}
	if r == "" {
		return fallbackEndpointLabel
	}
	return r
}
