package config

import (
	"sync"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

var (
	tokens     core.TokenMap[Config]
	tokensOnce sync.Once
)
