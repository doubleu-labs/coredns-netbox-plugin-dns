package plugin

import "github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"

var ContextKey = "netboxdns"

type ServerContext struct {
	APIClient *api.Client
}
