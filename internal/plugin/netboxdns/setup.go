package netboxdns

import (
	"net/url"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
)

func Register() {
	plugin.Register(PluginName, setup)
}

func setup(c *caddy.Controller) error {
	// TODO: implement

	u, err := url.Parse("https://localhost:9999/")
	if err != nil {
		return err
	}

	p := &netboxDNS{
		client: api.NewClient("token", u),
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
