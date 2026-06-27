package netboxdns_catalog

import (
	"context"
	"errors"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/miekg/dns"
)

const PluginName = "netboxdns.catalog"

type netboxDNSCatalog struct {
	Next   plugin.Handler
	logger log.P
	client *api.Client
}

func (*netboxDNSCatalog) Name() string {
	return PluginName
}

func (n *netboxDNSCatalog) ServeDNS(
	ctx context.Context,
	w dns.ResponseWriter,
	r *dns.Msg,
) (int, error) {
	if n.client == nil && api.NetboxClient != nil {
		n.client = api.NetboxClient
	}
	if n.client == nil {
		return dns.RcodeServerFailure, plugin.Error(
			PluginName,
			errors.New("netbox client not initialized"),
		)
	}

	return plugin.NextOrFailure(PluginName, n.Next, ctx, w, r)
}
