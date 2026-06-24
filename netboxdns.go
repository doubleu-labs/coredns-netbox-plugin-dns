package netboxdns

import (
	"context"
	"net/http"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/metrics"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/lookup"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/zonecache"
	"github.com/miekg/dns"
)

const (
	defaultHTTPClientTimeout time.Duration = time.Second * 5
	pluginName               string        = "netboxdns"

	defaultPollInterval time.Duration = 300 * time.Second
	defaultIXFRHistory  int           = 16
)

var logger log.P

func init() {
	logger = log.NewWithPlugin(pluginName)
}

type NetboxDNS struct {
	Next plugin.Handler

	requestClient *netbox.Client
	metrics       *metrics.Metrics

	zones       []string
	fall        fall.F
	viewName    string   // single-view (server-side filter)
	viewNames   []string // multi-view whitelist (client-side filter)
	viewExclude []string // view blacklist (client-side filter)

	// IXFR snapshot history. ixfrHistory == 0 disables the poller and the
	// IXFR delta path entirely (Transfer falls back to AXFR for stale
	// serials, which is the Phase 2 behaviour).
	pollInterval time.Duration
	ixfrHistory  int
	cache        *zonecache.Cache
	stopPoller   chan struct{}
	pollerDone   chan struct{}

	// Catalog zone state. catalogTracker is always non-nil so the poller
	// can call NextSerial without a guard; whether any catalog actually
	// exists is decided per poll cycle by GetCatalogZones.
	catalogTracker *catalogTracker
	catalogPrefix  string
}

func NewNetboxDNS() *NetboxDNS {
	return &NetboxDNS{
		requestClient: &netbox.Client{
			Client: &http.Client{
				Timeout: defaultHTTPClientTimeout,
			},
		},
		metrics:        metrics.NewMetrics(pluginName),
		zones:          []string{"."},
		pollInterval:   defaultPollInterval,
		ixfrHistory:    defaultIXFRHistory,
		catalogTracker: newCatalogTracker(),
	}
}

// Name implements the plugin.Handler interface
func (NetboxDNS) Name() string {
	return pluginName
}

// ServeDNS implements the plugin.Handler interface
func (netboxdns *NetboxDNS) ServeDNS(
	reqContext context.Context,
	respWriter dns.ResponseWriter,
	reqMsg *dns.Msg,
) (int, error) {
	state := request.Request{W: respWriter, Req: reqMsg}

	// Zone transfer requests must be handled by the transfer plugin, not
	// here. This guard ensures correct behavior regardless of plugin.cfg
	// ordering — if netboxdns is accidentally placed before transfer,
	// AXFR/IXFR queries are forwarded instead of producing a 400 from NetBox.
	if state.QType() == dns.TypeAXFR || state.QType() == dns.TypeIXFR {
		return netboxdns.nextOrFailure(reqContext, respWriter, reqMsg)
	}

	qname := state.QName()
	family := state.Family()
	qtype := fixQType(state.QType(), family)

	// check if plugin is configured to respond to the requested zone
	respondingZone := plugin.Zones(netboxdns.zones).Matches(qname)
	if respondingZone == "" {
		return netboxdns.nextOrFailure(reqContext, respWriter, reqMsg)
	}

	// Per-request metrics: latency is observed unconditionally, the
	// rcode counter is bumped at the single return points below via
	// recordServeResult so SERVFAIL paths are not lost.
	start := time.Now()
	defer func() {
		netboxdns.metrics.RequestDuration.WithLabelValues(respondingZone).
			Observe(time.Since(start).Seconds())
	}()

	l := &lookup.Lookup{
		Client:        netboxdns.requestClient,
		Logger:        &logger,
		QName:         qname,
		QType:         qtype,
		Family:        family,
		CatalogPrefix: netboxdns.catalogPrefix,
	}
	response, err := l.Run()
	if err != nil {
		netboxdns.metrics.RequestsTotal.WithLabelValues(
			respondingZone,
			rcodeLabel(dns.RcodeServerFailure),
		).Inc()
		return dns.RcodeServerFailure, err
	}
	if response.Result == lookup.NameError {
		if netboxdns.fall.Through(qname) {
			logger.Debugf(
				"forwarding request [%s] %q to next plugin",
				dns.TypeToString[qtype],
				qname,
			)
			return netboxdns.nextOrFailure(reqContext, respWriter, reqMsg)
		}
		logger.Debugf(
			"no records for [%s] %q; fallthrough not enabled",
			dns.TypeToString[qtype],
			qname,
		)

	}

	respMsg := &dns.Msg{
		Answer: response.Answer,
		Ns:     response.Ns,
		Extra:  response.Extra,
	}
	respMsg.SetReply(reqMsg)
	respMsg.Authoritative = true

	switch response.Result {
	case lookup.Success:
	case lookup.NameError:
		respMsg.Rcode = dns.RcodeNameError
	case lookup.Delegation:
		respMsg.Authoritative = false
	}

	netboxdns.metrics.RequestsTotal.WithLabelValues(
		respondingZone,
		rcodeLabel(respMsg.Rcode),
	).Inc()
	err = respWriter.WriteMsg(respMsg)
	if err != nil {
		return dns.RcodeServerFailure, err
	}
	return dns.RcodeSuccess, nil
}

// rcodeLabel renders a DNS rcode as a stable, low-cardinality label string.
func rcodeLabel(rcode int) string {
	if s, ok := dns.RcodeToString[rcode]; ok {
		return s
	}
	return "OTHER"
}

func (netboxdns *NetboxDNS) nextOrFailure(
	ctx context.Context,
	writer dns.ResponseWriter,
	request *dns.Msg,
) (int, error) {
	return plugin.NextOrFailure(
		pluginName,
		netboxdns.Next,
		ctx,
		writer,
		request,
	)
}

func fixQType(stateQtype uint16, family int) uint16 {
	var qtype uint16
	switch stateQtype {
	case dns.TypeA, dns.TypeAAAA:
		switch family {
		case 1:
			qtype = dns.TypeA
		case 2:
			qtype = dns.TypeAAAA
		}
		return qtype
	default:
		return stateQtype
	}
}
