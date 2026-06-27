package netboxdns

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/miekg/dns"
)

const PluginName = "netboxdns"

type netboxDNS struct {
	Next   plugin.Handler
	logger log.P
	client *api.Client
}

func (*netboxDNS) Name() string {
	return PluginName
}

func (n *netboxDNS) ServeDNS(
	ctx context.Context,
	w dns.ResponseWriter,
	r *dns.Msg,
) (int, error) {
	return plugin.NextOrFailure(PluginName, n.Next, ctx, w, r)
}
