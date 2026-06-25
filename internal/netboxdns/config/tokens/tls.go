package tokens

// noinspection LongLine
import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/tls"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/config"
)

var tTLSName = "tls"

func init() {
	config.RegisterToken(tTLSName, tTLS{})
}

type tTLS struct{}

func (t tTLS) Parse(c *caddy.Controller, cfg *config.Config) error {
	tc, err := tls.NewTLSConfigFromArgs(c.RemainingArgs()...)
	if err != nil {
		return config.ErrTokenParse(c, tTLSName, err)
	}
	cfg.TLS = tc
	return nil
}

func (t tTLS) Validate(_ *caddy.Controller, _ *config.Config) error {
	return nil
}
