package netboxdns_catalog

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
)

func Register() {
	plugin.Register(PluginName, setup)
}

func setup(c *caddy.Controller) error {
	// TODO: implement

	p := &netboxDNSCatalog{
		logger: log.NewWithPlugin(PluginName),
	}

	dnsserver.GetConfig(c).AddPlugin(
		func(next plugin.Handler) plugin.Handler {
			p.Next = next
			return p
		},
	)

	return nil
}
