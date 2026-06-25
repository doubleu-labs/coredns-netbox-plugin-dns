package tokens

// noinspection LongLine
import (
	"strconv"

	"github.com/coredns/caddy"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/config"
)

var (
	tIXFRHistoryName   = "ixfr_history"
	defaultIXFRHistory = 16
)

func init() {
	config.RegisterToken(tIXFRHistoryName, tIXFRHistory{})
}

type tIXFRHistory struct {
	set bool
}

func (t tIXFRHistory) Parse(c *caddy.Controller, cfg *config.Config) error {
	if !c.NextArg() {
		return config.ErrNoTokenValue(c, tIXFRHistoryName)
	}
	n, err := strconv.Atoi(c.Val())
	if err != nil {
		return config.ErrTokenParse(c, tIXFRHistoryName, err)
	}
	cfg.IXFRHistory = n
	t.set = true
	return nil
}

func (t tIXFRHistory) Validate(c *caddy.Controller, cfg *config.Config) error {
	if !t.set {
		cfg.IXFRHistory = defaultIXFRHistory
		return nil
	}
	if cfg.IXFRHistory < 0 {
		return config.ErrTokenValidate(
			c,
			tIXFRHistoryName,
			"must be greater than or equal to 0",
		)
	}
	return nil
}
