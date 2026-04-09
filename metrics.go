package netboxdns

import (
	"net/http"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics exported by the netboxdns plugin. Naming follows the
// CoreDNS convention `coredns_<plugin>_<thing>`. All collectors live in this
// file so the registration set is in one place; setup.go registers them
// once via promauto (registered on the default Prometheus registerer which
// the CoreDNS `prometheus` plugin scrapes).
var (
	// ----- DNS serving path -------------------------------------------------

	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "requests_total",
		Help:      "Counter of DNS requests handled by netboxdns, by zone and rcode.",
	}, []string{"zone", "rcode"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "request_duration_seconds",
		Help:      "Histogram of DNS request handling latency in netboxdns.",
		Buckets:   plugin.TimeBuckets,
	}, []string{"zone"})

	// ----- NetBox API client ----------------------------------------------

	netboxRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "netbox_requests_total",
		Help:      "Counter of HTTP requests to the NetBox API, by endpoint and HTTP code (\"error\" when the round-trip itself failed).",
	}, []string{"endpoint", "code"})

	netboxRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "netbox_request_duration_seconds",
		Help:      "Histogram of NetBox API round-trip latency, by endpoint.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 14),
	}, []string{"endpoint"})

	// ----- Outgoing zone transfers ----------------------------------------

	transfersTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "transfers_total",
		Help:      "Counter of outgoing zone transfers, by zone and kind (axfr, ixfr_delta, ixfr_noop, ixfr_fallback, catalog_axfr, catalog_ixfr_delta, catalog_ixfr_noop).",
	}, []string{"zone", "kind"})

	// ----- Poller ---------------------------------------------------------

	pollCyclesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "poll_cycles_total",
		Help:      "Counter of poller cycles, by result (success or error).",
	}, []string{"result"})

	pollDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "poll_duration_seconds",
		Help:      "Histogram of full poll-cycle duration.",
		Buckets:   prometheus.ExponentialBuckets(0.01, 2, 14),
	})

	// ----- Per-zone state -------------------------------------------------

	zoneSerial = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "zone_serial",
		Help:      "Last observed SOA serial of each zone served by netboxdns (catalog zones included).",
	}, []string{"zone"})

	cacheSnapshots = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "cache_snapshots",
		Help:      "Number of snapshots currently held in the IXFR ring buffer for each zone.",
	}, []string{"zone"})

	catalogMembers = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "catalog_members",
		Help:      "Number of member zones currently published in each catalog zone.",
	}, []string{"catalog"})
)

// instrumentedTransport wraps an http.RoundTripper and records per-request
// NetBox API metrics. It is installed on the plugin's http.Client in setup
// so internal/netbox stays free of metric imports.
type instrumentedTransport struct {
	base http.RoundTripper
}

func (t *instrumentedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	endpoint := netboxEndpointLabel(req.URL.Path)
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	netboxRequestDuration.WithLabelValues(endpoint).Observe(time.Since(start).Seconds())
	if err != nil {
		netboxRequestsTotal.WithLabelValues(endpoint, "error").Inc()
		return resp, err
	}
	netboxRequestsTotal.WithLabelValues(endpoint, http.StatusText(resp.StatusCode)).Inc()
	return resp, nil
}

// netboxEndpointLabel reduces a NetBox API path to a low-cardinality label.
// The netbox-dns plugin paths look like /api/plugins/netbox-dns/<thing>/...
// — we keep just <thing> (zones, records, nameservers, views, ...). Anything
// outside that shape collapses to "other".
func netboxEndpointLabel(path string) string {
	const prefix = "/api/plugins/netbox-dns/"
	if !strings.HasPrefix(path, prefix) {
		return "other"
	}
	rest := strings.TrimPrefix(path, prefix)
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		rest = rest[:i]
	}
	if rest == "" {
		return "other"
	}
	return rest
}
