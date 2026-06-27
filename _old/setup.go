package netboxdns

// noinspection LongLine
import (
	"fmt"
	"slices"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/metrics"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netboxdns/config"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/view"
)

func init() {
	plugin.Register(pluginName, setup)
}

func setup(controller *caddy.Controller) error {
	cfg, err := config.Parse(controller)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	m := metrics.NewMetrics(pluginName)
	rq := netbox.NewClient(m)
	if cfg.Timeout != 0 {
		rq.SetTimeout(cfg.Timeout)
	}
	if cfg.TLS != nil {
		rq.SetTLSConfig(cfg.TLS)
	}
	rq.SetToken(cfg.Token)
	rq.NetboxURL = cfg.URL.JoinPath("api", "plugins", "netbox-dns")
	rq.UserAgent = fmt.Sprintf("coredns plugin %s", pluginName)
	v := view.New(cfg.Views, cfg.ViewsExclude)
	netboxdns := &NetboxDNS{
		requestClient: rq,
		metrics:       m,
		cache:         catalog_old.NewCache(cfg.IXFRHistory),
		poller: catalog_old.NewPoller(
			&logger,
			m,
			rq,
			v,
			&cfg.CatalogPrefix,
		),
		fall:         cfg.Fall,
		ixfrHistory:  cfg.IXFRHistory,
		pollInterval: cfg.PollInterval,
		views:        v,
		zones:        cfg.Zones,
	}

	// instead of failing fast on an unknown view, log a warning containing the
	// unknown view names and purge them from specified views to prevent
	// SERVFAIL errors while keeping the server online.
	if uv, err := netboxdns.processViews(); err != nil {
		return plugin.Error(
			pluginName,
			fmt.Errorf(
				"[setup] error getting views: %w",
				err,
			),
		)
	} else if len(uv) != 0 {
		logger.Warningf(
			"[setup] ignoring views not found in netbox: %v",
			uv,
		)
	}

	dnsserver.GetConfig(controller).AddPlugin(
		func(next plugin.Handler) plugin.Handler {
			netboxdns.Next = next
			return netboxdns
		},
	)

	controller.OnStartup(
		func() error {
			netboxdns.poller.Start()
			return nil
		},
	)
	controller.OnShutdown(
		func() error {
			netboxdns.poller.StopAndWait()
			return nil
		},
	)

	logger.Info("successfully started netboxdns")
	return nil
}

func (n *NetboxDNS) processViews() ([]string, error) {
	_, uv, err := netbox.GetViews(n.requestClient, n.views)
	if err != nil {
		return nil, err
	}
	if len(uv) != 0 {
		for _, v := range uv {
			n.views.Include = slices.DeleteFunc(
				n.views.Include,
				func(s string) bool {
					return s == v
				},
			)
			n.views.Exclude = slices.DeleteFunc(
				n.views.Exclude,
				func(s string) bool {
					return s == v
				},
			)
		}
		return uv, nil
	}
	return nil, nil
}
