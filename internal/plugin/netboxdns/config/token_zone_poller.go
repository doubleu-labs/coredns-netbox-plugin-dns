package config

import (
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func init() {
	core.RegisterToken(
		&tokensOnce,
		&tokens,
		tZonePollerName,
		new(tZonePoller),
	)
}

const tZonePollerName = "zone_poller"

type tZonePoller struct{}

func (tZonePoller) Parse(c *caddy.Controller, cfg *Config) error {
	if !c.NextArg() {
		cfg.ZonePollerEnabled = true
		return nil
	}
	d, err := time.ParseDuration(c.Val())
	if err != nil {
		return core.ErrTokenParse(c, tZonePollerName, err)
	}
	cfg.ZonePollerDuration = d
	cfg.ZonePollerEnabled = true
	return nil
}

func (tZonePoller) Validate(_ *caddy.Controller, _ log.P, _ *Config) error {
	return nil
}
