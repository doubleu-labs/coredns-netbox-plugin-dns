package netboxdns

//noinspection LongLine
import (
	"testing"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns_catalog"
)

func init() {
	if testing.Testing() {
		netboxdns.SetTestRegister()
	}
	netboxdns.Register()
	netboxdns_catalog.Register()
	core.FinalizePlugins()
}
