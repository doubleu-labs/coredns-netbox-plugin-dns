package netboxdns

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coredns/caddy"
)

// newSetupTestServer returns a httptest.Server that pretends to be a NetBox
// instance for setup-time view validation. The handler returns 200 with an
// empty zone list when ?view=<wantView> is passed and 400 (mirroring real
// NetBox behaviour) for any other view value.
func newSetupTestServer(t *testing.T, wantView string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/plugins/netbox-dns/zones/", func(w http.ResponseWriter, r *http.Request) {
		got := r.URL.Query().Get("view")
		if got != wantView {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"view": ["Select a valid choice."]}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"count": 0, "next": null, "previous": null, "results": []}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestSetup_ViewValidation_Success(t *testing.T) {
	srv := newSetupTestServer(t, "internal")
	corefile := fmt.Sprintf(`netboxdns {
		token sometoken
		url %s
		view internal
	}`, srv.URL)

	controller := caddy.NewTestController("dns", corefile)
	if err := setup(controller); err != nil {
		t.Fatalf("setup with valid view: unexpected error: %v", err)
	}
}

func TestSetup_ViewValidation_RejectsUnknownView(t *testing.T) {
	srv := newSetupTestServer(t, "internal")
	corefile := fmt.Sprintf(`netboxdns {
		token sometoken
		url %s
		view typo-internl
	}`, srv.URL)

	controller := caddy.NewTestController("dns", corefile)
	err := setup(controller)
	if err == nil {
		t.Fatal("expected setup to fail for unknown view, got nil")
	}
	if !strings.Contains(err.Error(), "typo-internl") {
		t.Errorf("error message should mention the bad view name; got: %v", err)
	}
}

func TestSetup_NoView_SkipsValidation(t *testing.T) {
	// No mock server: setup must succeed without making any HTTP call when
	// view is not configured. We point url at a guaranteed-dead address to
	// prove validation is not happening.
	corefile := `netboxdns {
		token sometoken
		url http://127.0.0.1:1
	}`
	controller := caddy.NewTestController("dns", corefile)
	if err := setup(controller); err != nil {
		t.Fatalf("setup without view should not validate; got: %v", err)
	}
}
