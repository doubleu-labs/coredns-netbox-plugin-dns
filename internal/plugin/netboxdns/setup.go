package netboxdns

// noinspection LongLine
import (
	"fmt"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	iplugin "github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns/config"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns/poller"
)

var (
	isSetupTest    bool
	isTestRegister bool
)

func SetTestRegister() {
	isTestRegister = true
}

func Register() {
	if isTestRegister {
		core.RegisterPlugin(pluginName, setup).After("transfer")
		return
	}
	core.RegisterPlugin(pluginName, setup)
}

func setup(c *caddy.Controller) error {
	logger := log.NewWithPlugin(pluginName)

	cfg, err := config.Parse(c, logger)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	ctx := getServerContext(c, cfg)

	views := &core.Views{
		Include: cfg.Views,
		Exclude: cfg.ViewsExclude,
	}

	netboxdns := &netboxDNS{
		activeZoneStatus: cfg.ActiveZoneStatus,
		client:           ctx.APIClient,
		logger:           logger,
		noop:             cfg.NoOp,
		serverState:      ctx,
		views:            views,
		zones:            cfg.Zones,
	}

	if len(views.Include) > 0 || len(views.Exclude) > 0 {
		viewPoller, vpErr := poller.NewViewPoller(
			ctx.APIClient,
			views,
			cfg.ViewPollerDuration,
		)
		if vpErr != nil {
			return vpErr
		}
		netboxdns.viewPoller = viewPoller
		c.OnStartup(
			func() error {
				viewPoller.Start()
				return nil
			},
		)
		c.OnShutdown(
			func() error {
				viewPoller.Stop()
				return nil
			},
		)
	}

	if cfg.Fall != nil {
		netboxdns.fall = *cfg.Fall
	}

	dnsserver.GetConfig(c).AddPlugin(
		func(next plugin.Handler) plugin.Handler {
			netboxdns.Next = next
			return netboxdns
		},
	)

	return nil
}

func getServerContext(
	c *caddy.Controller,
	cfg *config.Config,
) *iplugin.ServerContext {
	var ctx *iplugin.ServerContext
	if value, ok := c.Get(iplugin.ContextKey).(*iplugin.ServerContext); ok {
		ctx = value
	} else {
		ctx = &iplugin.ServerContext{
			APIClient: api.NewClient(cfg.Token, cfg.NetboxURL),
		}
		if cfg.Timeout != 0 {
			ctx.APIClient.SetTimeout(cfg.Timeout)
		}
		if cfg.TLS != nil {
			ctx.APIClient.SetTLSConfig(cfg.TLS)
		}
		ctx.APIClient.SetUserAgent(
			fmt.Sprintf("coredns.plugin.%s", pluginName),
		)
	}
	return ctx
}
