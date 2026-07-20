package netboxdns

import (
	"os"
	"testing"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func TestMain(m *testing.M) {
	SetTestRegister()
	Register()
	core.FinalizePlugins()
	os.Exit(m.Run())
}
