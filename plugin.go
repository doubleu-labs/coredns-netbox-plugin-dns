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

func ensureDirectiveExecutionOrder(directives []string) {
	pluginCoreIndex := slices.Index(directives, netboxdns.PluginName)
	if pluginCoreIndex == -1 {
		return
	}
	dnsserver.Directives = slices.Insert(
		directives,
		pluginCoreIndex,
		netboxdns_catalog.PluginName,
	)
}
