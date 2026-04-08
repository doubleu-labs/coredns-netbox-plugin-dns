package netboxdns

import (
	"fmt"
	"net/http"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
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
	netboxdns.requestClient.Client.Transport = &instrumentedTransport{base: base}
	if netboxdns.viewName != "" {
		// Fail fast on misconfiguration: NetBox returns HTTP 400 when an
		// unknown view name is passed to /zones/?view=, so without this
		// check every DNS query would silently produce SERVFAIL.
		if _, err := netbox.GetZones(netboxdns.requestClient, netboxdns.viewName); err != nil {
			return plugin.Error(pluginName, fmt.Errorf(
				"validating netbox view %q: %w", netboxdns.viewName, err))
		}
	}
	dnsserver.GetConfig(controller).AddPlugin(
		func(next plugin.Handler) plugin.Handler {
			netboxdns.Next = next
			return netboxdns
		},
	)

	controller.OnStartup(func() error {
		netboxdns.startPoller()
		return nil
	})
	controller.OnShutdown(func() error {
		netboxdns.stopPollerAndWait()
		return nil
	})

	logger.Info("successfully started netboxdns")
	return nil
}
