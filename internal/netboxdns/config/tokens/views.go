package tokens

// noinspection LongLine
import (
	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/config"
)

var tViewsName = "views"

func init() {
	config.RegisterToken(tViewsName, tViews{})
}

type tViews struct{}

func (tViews) Parse(c *caddy.Controller, cfg *config.Config) error {
	a := c.RemainingArgs()
	if len(a) == 0 {
		return config.ErrNoTokenValue(c, tViewsName)
	}
	cfg.Views = a
	return nil
}

func (tViews) Validate(_ *caddy.Controller, _ *config.Config) error {
	return nil
}
