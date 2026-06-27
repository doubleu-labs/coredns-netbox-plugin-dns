package config

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/fall"
)

type Config struct {
	CatalogPrefix string        `name:"catalog_prefix"`
	Fall          fall.F        `name:"fallthrough"`
	IXFRHistory   int           `name:"ixfr_history"`
	PollInterval  time.Duration `name:"poll_interval"`
	Timeout       time.Duration `name:"timeout"`
	TLS           *tls.Config   `name:"tls"`
	Token         string        `name:"token" required:"true"`
	URL           *url.URL      `name:"url" required:"true"`
	Views         []string      `name:"views"`
	ViewsExclude  []string      `name:"views_exclude"`
	Zones         []string
}

func (cfg *Config) validate(c *caddy.Controller) error {
	t := reflect.TypeOf(*cfg)
	v := reflect.ValueOf(*cfg)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tagRequired := field.Tag.Get("required")
		if tagRequired == "" {
			continue
		}
		isRequired, err := strconv.ParseBool(tagRequired)
		if err != nil {
			continue
		}
		if isRequired && v.Field(i).IsZero() {
			tagName := field.Tag.Get("name")
			return ErrTokenValidate(c, tagName, "required")
		}
	}
	return nil
}

type Token interface {
	Parse(*caddy.Controller, *Config) error
	Validate(*caddy.Controller, *Config) error
}

type tokenMap map[string]Token

func (tm tokenMap) tokens() string {
	var e string
	var i int
	for t := range tm {
		e += fmt.Sprintf("%q", t)
		if i+1 < len(tm) {
			e += ", "
		}
		if i == len(tm)-2 {
			e += "or "
		}
		i++
	}
	return e
}
func (tm tokenMap) unknownToken(c *caddy.Controller, t string) error {
	return c.Errf(
		"[config] unknown token %q; expected %s",
		t,
		tm.tokens(),
	)
}

func (tm tokenMap) parse(c *caddy.Controller, cfg *Config) error {
	for c.NextBlock() {
		tn := c.Val()
		t, ok := tm[tn]
		if !ok {
			return tm.unknownToken(c, tn)
		}
		if err := t.Parse(c, cfg); err != nil {
			return err
		}
	}
	return nil
}

func (tm tokenMap) validate(c *caddy.Controller, cfg *Config) error {
	for tn, t := range tm {
		if err := t.Validate(c, cfg); err != nil {
			return c.Errf(
				"[config] error validating token %q: %v",
				tn,
				err,
			)
		}
	}
	return nil
}

var (
	tokensOnce sync.Once
	tokens     tokenMap
)

func RegisterToken(n string, t Token) {
	tokensOnce.Do(
		func() {
			if tokens == nil {
				tokens = make(tokenMap)
			}
		},
	)
	tokens[n] = t
}
