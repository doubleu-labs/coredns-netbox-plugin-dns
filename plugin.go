package netboxdns

// noinspection LongLine
import (
	"slices"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns_catalog"
)

func init() {
	netboxdns.Register()
	netboxdns_catalog.Register()
	ensureDirectiveExecutionOrder(dnsserver.Directives)
}

func ensureDirectiveExecutionOrder(ds []string) {
	coreIdx := slices.Index(ds, netboxdns.PluginName)
	if coreIdx == -1 {
		return
	}
	dnsserver.Directives = slices.Insert(
		ds,
		coreIdx,
		netboxdns_catalog.PluginName,
	)
}
