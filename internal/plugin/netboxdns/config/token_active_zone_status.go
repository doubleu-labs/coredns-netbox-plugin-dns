package config

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func init() {
	core.RegisterToken(
		&tokensOnce,
		&tokens,
		tActiveZoneStatusName,
		new(tActiveZoneStatus),
	)
}

const tActiveZoneStatusName = "active_zone_status"

type tActiveZoneStatus struct{}

func (tActiveZoneStatus) Parse(c *caddy.Controller, cfg *Config) error {
	args := c.RemainingArgs()
	if len(args) == 0 {
		return core.ErrNoTokenValue(c, tActiveZoneStatusName)
	}
	if len(cfg.ActiveZoneStatus) != 0 {
		cfg.ActiveZoneStatus = append(cfg.ActiveZoneStatus, args...)
		return nil
	}
	cfg.ActiveZoneStatus = args
	return nil
}

func (tActiveZoneStatus) Validate(
	_ *caddy.Controller,
	l log.P,
	cfg *Config,
) error {
	if len(cfg.ActiveZoneStatus) == 0 {
		cfg.ActiveZoneStatus = []string{"active", "dynamic"}
	}
	return nil
}
