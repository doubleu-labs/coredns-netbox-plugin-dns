package netboxdns

import "testing"

func TestNetboxEndpointLabel(t *testing.T) {
	cases := map[string]string{
		"/api/plugins/netbox-dns/zones/":           "zones",
		"/api/plugins/netbox-dns/zones/7/":         "zones",
		"/api/plugins/netbox-dns/records/?zone=3":  "records",
		"/api/plugins/netbox-dns/nameservers/":     "nameservers",
		"/api/plugins/netbox-dns/":                 "other",
		"/api/dcim/devices/":                       "other",
		"":                                         "other",
	}
	for in, want := range cases {
		if got := netboxEndpointLabel(in); got != want {
			t.Errorf("netboxEndpointLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
