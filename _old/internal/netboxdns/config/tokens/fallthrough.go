package tokens

// noinspection LongLine
import (
	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/config"
)

func init() {
	config.RegisterToken(tFallthroughName, tFallthrough{})
}

var tFallthroughName = "fallthrough"

type tFallthrough struct{}

func (tFallthrough) Parse(c *caddy.Controller, cfg *config.Config) error {
	cfg.Fall.SetZonesFromArgs(c.RemainingArgs())
	return nil
}

func (tFallthrough) Validate(_ *caddy.Controller, _ *config.Config) error {
	return nil
}
