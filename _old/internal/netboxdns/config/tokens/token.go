package tokens

// noinspection LongLine
import (
	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/config"
)

var tTokenName = "token"

func init() {
	config.RegisterToken(tTokenName, tToken{})
}

type tToken struct{}

func (tToken) Parse(c *caddy.Controller, cfg *config.Config) error {
	if !c.NextArg() {
		return config.ErrNoTokenValue(c, tTokenName)
	}
	cfg.Token = c.Val()
	return nil
}

func (tToken) Validate(_ *caddy.Controller, _ *config.Config) error {
	return nil
}
