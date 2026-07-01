package config

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

// Token `noop` configures the plugin to not return any records. Must be used in
// conjunction with `fallthrough`. If not used with fallthrough, the plugin
// returns SERVFAIL. Only really useful for when other sub-plugins are needed,
// but not the core functionality.

func init() {
	core.RegisterToken[Config](
		&tokensOnce,
		&tokens,
		tNoOpName,
		&tNoOp{},
	)
}

var tNoOpName = "noop"

type tNoOp struct{}

func (tNoOp) Parse(_ *caddy.Controller, cfg *Config) error {
	cfg.NoOp = true
	return nil
}

func (tNoOp) Validate(_ *caddy.Controller, _ log.P, _ *Config) error {
	return nil
}
