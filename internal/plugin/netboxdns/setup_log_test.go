package netboxdns

import (
	"testing"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
)

func Test_SetupFallthrough(t *testing.T) {
	tt := []testutil.SetupTest{
		{
			Name: "fallthrough unreachable plugin scope",
			ServerBlock: newValidTestConfig().
				set("zones", "example.com").
				set("fallthrough", "example.net").
				format(),
			Want: "[WARNING] plugin/netboxdns: [config] `fallthrough` zones " +
				"[example.net.] are not reachable due to server and plugin " +
				"zones: [example.com.]",
		},
		{
			Name: "fallthrough unreachable server scope",
			ServerBlock: newValidTestConfig().
				set("fallthrough", "example.net").
				format(),
			ServerBlockKeys: []string{"example.com."},
			Want: "[WARNING] plugin/netboxdns: [config] `fallthrough` zones " +
				"[example.net.] are not reachable due to server and plugin " +
				"zones: [example.com.]",
		},
	}

	testutil.RunSetupLogTests(t, tt, setup, &isSetupTest)
}
