package tokens

// noinspection LongLine
import (
	"time"

	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/config"
)

var tTimeoutName = "timeout"

func init() {
	config.RegisterToken(tTimeoutName, tTimeout{})
}

type tTimeout struct{}

func (tTimeout) Parse(c *caddy.Controller, cfg *config.Config) error {
	if !c.NextArg() {
		return config.ErrNoTokenValue(c, tTimeoutName)
	}
	d, err := time.ParseDuration(c.Val())
	if err != nil {
		return config.ErrTokenParse(c, tTimeoutName, err)
	}
	cfg.Timeout = d
	return nil
}

func (tTimeout) Validate(_ *caddy.Controller, _ *config.Config) error {
	return nil
}
