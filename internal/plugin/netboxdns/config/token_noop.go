package config

import (
	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

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

func (*tNoOp) Parse(_ *caddy.Controller, cfg *Config) error {
	cfg.NoOp = true
	return nil
}

func (*tNoOp) Validate(_ *caddy.Controller, _ *Config) error {
	return nil
}
