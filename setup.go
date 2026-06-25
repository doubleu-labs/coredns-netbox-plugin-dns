package netboxdns

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/metrics"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
)

func init() {
	plugin.Register(pluginName, setup)
}

func setup(controller *caddy.Controller) error {
	netboxdns := NewNetboxDNS()
	if err := Parse(controller, netboxdns); err != nil {
		return err
	}
	// Wrap the HTTP client transport so every NetBox API round-trip is
	// observed by the netbox_requests_total / netbox_request_duration
	// collectors. Done after Parse so the tls{} block has had a chance to
	// install its own base transport.
	base := netboxdns.requestClient.Client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	netboxdns.requestClient.Client.Transport =
		metrics.NewInstrumentedTransport(
			base,
			netboxdns.metrics,
		)

	// instead of failing fast on an unknown view, log a warning containing the
	// unknown view names and purge them from specified views to prevent
	// SERVFAIL errors while keeping the server online.
	if uv, err := netboxdns.processViews(); err != nil {
		return plugin.Error(
			pluginName,
			fmt.Errorf(
				"[setup] error getting views: %w",
				err,
			),
		)
	} else if len(uv) != 0 {
		logger.Warningf(
			"[setup] ignoring views not found in netbox: %v",
			uv,
		)
	}

	dnsserver.GetConfig(controller).AddPlugin(
		func(next plugin.Handler) plugin.Handler {
			netboxdns.Next = next
			return netboxdns
		},
	)

	controller.OnStartup(
		func() error {
			netboxdns.poller.Start()
			return nil
		},
	)
	controller.OnShutdown(
		func() error {
			netboxdns.poller.StopAndWait()
			return nil
		},
	)

	logger.Info("successfully started netboxdns")
	return nil
}

func (n *NetboxDNS) processViews() ([]string, error) {
	_, uv, err := netbox.GetViews(n.requestClient, n.views)
	if err != nil {
		return nil, err
	}
	if len(uv) != 0 {
		for _, v := range uv {
			n.views.Include = slices.DeleteFunc(
				n.views.Include,
				func(s string) bool {
					return s == v
				},
			)
			n.views.Exclude = slices.DeleteFunc(
				n.views.Exclude,
				func(s string) bool {
					return s == v
				},
			)
		}
		return uv, nil
	}
	return nil, nil
}
