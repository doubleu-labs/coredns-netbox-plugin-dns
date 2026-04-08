package netboxdns

import (
	"fmt"

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
	logger.Info("successfully started netboxdns")
	return nil
}
