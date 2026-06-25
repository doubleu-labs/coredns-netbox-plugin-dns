package tokens

// noinspection LongLine
import (
	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/config"
)

var tViewsExcludeName = "view_exclude"

func init() {
	config.RegisterToken(tViewsExcludeName, tViewsExclude{})
}

type tViewsExclude struct{}

func (tViewsExclude) Parse(c *caddy.Controller, cfg *config.Config) error {
	a := c.RemainingArgs()
	if len(a) == 0 {
		return config.ErrNoTokenValue(c, tViewsExcludeName)
	}
	cfg.ViewsExclude = a
	return nil
}

func (tViewsExclude) Validate(_ *caddy.Controller, _ *config.Config) error {
	return nil
}
