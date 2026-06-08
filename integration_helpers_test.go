package netboxdns

import (
	"net"
	"sync"
	"testing"
	"time"
)

// requireNetbox is called by tests that need a live NetBox instance reachable
// at testInstanceUrlHost (localhost:9999 by default, brought up via
// `just instance-start`). When the host is unreachable the test is skipped
// rather than failed, so `go test ./...` works on a developer machine that
// has not started the Podman test instance.
//
// The probe runs once per test process; the result is cached.
func requireNetbox(t *testing.T) {
	t.Helper()
	netboxProbeOnce.Do(func() {
		conn, err := net.DialTimeout("tcp", testInstanceUrlHost, 300*time.Millisecond)
		if err != nil {
			netboxProbeErr = err
			return
		}
		_ = conn.Close()
	})
	if netboxProbeErr != nil {
		t.Skipf("netbox integration test: %s unreachable (%v); start it with `just instance-start`",
			testInstanceUrlHost, netboxProbeErr)
	}
}

var (
	netboxProbeOnce sync.Once
	netboxProbeErr  error
)
