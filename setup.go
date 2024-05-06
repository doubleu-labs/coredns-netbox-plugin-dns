package netboxdns

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() {
	plugin.Register(pluginName, setup)
}

func setup(controller *caddy.Controller) error {
	netboxdns := NewNetboxDNS()
	if err := Parse(controller, netboxdns); err != nil {
		return err
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
