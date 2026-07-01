package config

import (
	"fmt"
	"slices"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/miekg/dns"
)

// Token `fallthrough` configures the plugin to pass requests on to the next
// plugin in the chain if no records are found.

func init() {
	core.RegisterToken[Config](
		&tokensOnce,
		&tokens,
		tFallthroughName,
		&tFallthrough{},
	)
}

var tFallthroughName = "fallthrough"

type tFallthrough struct{}

func (tFallthrough) Parse(c *caddy.Controller, cfg *Config) error {
	cfg.Fall = &fall.F{}
	cfg.Fall.SetZonesFromArgs(c.RemainingArgs())
	return nil
}

func (tFallthrough) Validate(_ *caddy.Controller, l log.P, cfg *Config) error {
	if cfg.Fall == nil {
		return nil
	}
	invalid := make([]string, 0)
	if !cfg.Fall.Equal(fall.Zero) && len(cfg.Fall.Zones) > 0 {
		for fallZone := range slices.Values(cfg.Fall.Zones) {
			valid := false
			for pluginZone := range slices.Values(cfg.Zones) {
				if dns.IsSubDomain(pluginZone, fallZone) {
					valid = true
					break
				}
			}
			if !valid {
				invalid = append(invalid, fallZone)
			}
		}
	}
	if len(invalid) > 0 {
		l.Warning(
			core.ScopedMessage(
				"config",
				fmt.Sprintf(
					"`%s` zones %v are not reachable due to server "+
						"and plugin zones: %v",
					tFallthroughName,
					invalid,
					cfg.Zones,
				),
			),
		)
	}
	return nil
}
