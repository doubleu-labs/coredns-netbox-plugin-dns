package netboxdns

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
)

// ----- parseView / parseViewExclude ----------------------------------------

func TestParseView_MultipleValues(t *testing.T) {
	corefile := `netboxdns {
		token t
		url http://127.0.0.1:1
		view internal external
	}`
	n := NewNetboxDNS()
	c := caddy.NewTestController("dns", corefile)
	c.Next()
	parseZones(c, n)
	if err := parseConfigTokens(c, n); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(n.viewNames) != 2 || n.viewNames[0] != "internal" || n.viewNames[1] != "external" {
		t.Errorf("viewNames = %v, want [internal external]", n.viewNames)
	}
	// single-view shortcut should NOT be set for multi-view
	if n.viewName != "" {
		t.Errorf("viewName should be empty for multi-view, got %q", n.viewName)
	}
}

func TestParseView_SingleValueSetsViewName(t *testing.T) {
	corefile := `netboxdns {
		token t
		url http://127.0.0.1:1
		view external
	}`
	n := NewNetboxDNS()
	c := caddy.NewTestController("dns", corefile)
	c.Next()
	parseZones(c, n)
	if err := parseConfigTokens(c, n); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if n.viewName != "external" {
		t.Errorf("viewName = %q, want external", n.viewName)
	}
	if len(n.viewNames) != 1 {
		t.Errorf("viewNames len = %d, want 1", len(n.viewNames))
	}
}

func TestParseViewExclude(t *testing.T) {
	corefile := `netboxdns {
		token t
		url http://127.0.0.1:1
		view_exclude staging test
	}`
	n := NewNetboxDNS()
	c := caddy.NewTestController("dns", corefile)
	c.Next()
	parseZones(c, n)
	if err := parseConfigTokens(c, n); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(n.viewExclude) != 2 || n.viewExclude[0] != "staging" || n.viewExclude[1] != "test" {
		t.Errorf("viewExclude = %v, want [staging test]", n.viewExclude)
	}
}

func TestParseView_MutuallyExclusiveWithExclude(t *testing.T) {
	corefile := `netboxdns {
		token t
		url http://127.0.0.1:1
		view_exclude staging
		view external
	}`
	n := NewNetboxDNS()
	c := caddy.NewTestController("dns", corefile)
	c.Next()
	parseZones(c, n)
	err := parseConfigTokens(c, n)
	if err == nil {
		t.Fatal("expected mutual-exclusion error, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error = %v, should mention mutually exclusive", err)
	}
}

func TestParseViewExclude_MutuallyExclusiveWithView(t *testing.T) {
	corefile := `netboxdns {
		token t
		url http://127.0.0.1:1
		view external
		view_exclude staging
	}`
	n := NewNetboxDNS()
	c := caddy.NewTestController("dns", corefile)
	c.Next()
	parseZones(c, n)
	err := parseConfigTokens(c, n)
	if err == nil {
		t.Fatal("expected mutual-exclusion error, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error = %v, should mention mutually exclusive", err)
	}
}

// ----- filterZonesByView ---------------------------------------------------

func zones(names ...string) []netbox.Zone {
	out := make([]netbox.Zone, len(names))
	for i, n := range names {
		parts := strings.SplitN(n, "@", 2)
		out[i] = netbox.Zone{ID: i + 1, Name: parts[0]}
		if len(parts) == 2 {
			out[i].View = &netbox.View{Name: parts[1]}
		}
	}
	return out
}

func TestFilterZonesByView_Whitelist(t *testing.T) {
	n := &NetboxDNS{viewNames: []string{"internal", "external"}}
	in := zones("a.com@internal", "b.com@external", "c.com@staging", "d.com@internal")
	got := n.filterZonesByView(in)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	names := make([]string, len(got))
	for i := range got {
		names[i] = got[i].Name
	}
	want := "a.com,b.com,d.com"
	if strings.Join(names, ",") != want {
		t.Errorf("got %v, want %s", names, want)
	}
}

func TestFilterZonesByView_Blacklist(t *testing.T) {
	n := &NetboxDNS{viewExclude: []string{"staging"}}
	in := zones("a.com@internal", "b.com@staging", "c.com@external")
	got := n.filterZonesByView(in)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	for _, z := range got {
		if z.View != nil && strings.EqualFold(z.View.Name, "staging") {
			t.Errorf("staging zone should be excluded: %v", z)
		}
	}
}

func TestFilterZonesByView_NoFilter(t *testing.T) {
	n := &NetboxDNS{}
	in := zones("a.com@v1", "b.com@v2")
	got := n.filterZonesByView(in)
	if len(got) != 2 {
		t.Errorf("len = %d, want 2 (pass-through)", len(got))
	}
}

func TestFilterZonesByView_NilView(t *testing.T) {
	n := &NetboxDNS{viewNames: []string{"internal"}}
	in := []netbox.Zone{{ID: 1, Name: "a.com"}} // View == nil
	got := n.filterZonesByView(in)
	if len(got) != 0 {
		t.Errorf("zone with nil view should not match whitelist, got %v", got)
	}
}

// ----- setup multi-view validation -----------------------------------------

func newMultiViewTestServer(t *testing.T, validViews map[string]bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/plugins/netbox-dns/zones/", func(w http.ResponseWriter, r *http.Request) {
		got := r.URL.Query().Get("view")
		if !validViews[got] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"view": ["Select a valid choice."]}`))
			return
		}
		writeJSON(w, `{"count": 0, "next": null, "previous": null, "results": []}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestSetup_MultiView_AllValid(t *testing.T) {
	srv := newMultiViewTestServer(t, map[string]bool{"internal": true, "external": true})
	corefile := fmt.Sprintf(`netboxdns {
		token t
		url %s
		view internal external
	}`, srv.URL)
	c := caddy.NewTestController("dns", corefile)
	if err := setup(c); err != nil {
		t.Fatalf("setup: %v", err)
	}
}

func TestSetup_MultiView_OneInvalid(t *testing.T) {
	srv := newMultiViewTestServer(t, map[string]bool{"internal": true})
	corefile := fmt.Sprintf(`netboxdns {
		token t
		url %s
		view internal typo
	}`, srv.URL)
	c := caddy.NewTestController("dns", corefile)
	err := setup(c)
	if err == nil {
		t.Fatal("expected error for invalid view")
	}
	if !strings.Contains(err.Error(), "typo") {
		t.Errorf("error should mention bad view: %v", err)
	}
}
