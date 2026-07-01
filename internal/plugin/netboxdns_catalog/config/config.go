package config

import "time"

type config struct {
	CatalogPrefix string        `name:"catalog_prefix"`
	IXFRHistory   int           `name:"ixfr_history"`
	PollInterval  time.Duration `name:"poll_interval"`
	Views         []string      `name:"views"`
	ViewsExclude  []string      `name:"views_exclude"`
	Zones         []string
}
