package tokens

// noinspection LongLine
import (
	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/config"
)

var tCatalogPrefixName = "catalog_prefix"

func init() {
	config.RegisterToken(tCatalogPrefixName, tCatalogPrefix{})
}

type tCatalogPrefix struct{}

func (tCatalogPrefix) Parse(c *caddy.Controller, cfg *config.Config) error {
	if !c.NextArg() {
		return config.ErrNoTokenValue(c, tCatalogPrefixName)
	}
	cfg.CatalogPrefix = c.Val()
	return nil
}

func (tCatalogPrefix) Validate(_ *caddy.Controller, _ *config.Config) error {
	return nil
}
