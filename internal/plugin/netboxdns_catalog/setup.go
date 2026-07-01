package netboxdns_catalog

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func Register() {
	core.RegisterPlugin(pluginName, setup).Before("netboxdns")
}

func setup(c *caddy.Controller) error {
	// TODO: implement

	p := &netboxDNSCatalog{
		logger: log.NewWithPlugin(pluginName),
	}

	dnsserver.GetConfig(c).AddPlugin(
		func(next plugin.Handler) plugin.Handler {
			p.Next = next
			return p
		},
	)

	return nil
}
