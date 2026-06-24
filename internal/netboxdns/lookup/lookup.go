package lookup

import (
	"slices"
	"strings"
	"sync"

	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/zonecache"
	"github.com/miekg/dns"
)

type Lookup struct {
	Client        *netbox.Client
	Logger        *log.P
	QName         string
	QType         uint16
	Family        int
	ViewNames     []string
	ViewExclude   []string
	CatalogPrefix string
	Cache         *zonecache.Cache

	settledOnce  sync.Once
	settledViews []string
}

func (l *Lookup) Run() (*Response, error) {
	nt := strings.TrimSuffix(l.QName, ".")
	l.settleViews()

	// gather catalog zones
	if cz, err := l.catalogZones(nt); err != nil {
		return nil, err
	} else if cz != nil {
		return cz, nil
	}

	// check if zone exists and is active
	z, err := l.matchZone(nt)
	if err != nil {
		return nil, err
	}
	if z == nil {
		l.Logger.Debugf("no zone matching %q", l.QName)
		return &Response{Result: NameError}, nil
	}

	// if the query is for the zone origin, return origin records
	o, err := l.processOrigin(z, nt)
	if err != nil {
		return nil, err
	}
	if o != nil {
		l.Logger.Debugf(
			"found origin records for [%s] %q",
			dns.TypeToString[l.QType],
			l.QName,
		)
		return o, nil
	}

	// look up exact request
	d, err := l.direct(z)
	if err != nil {
		return nil, err
	}
	if d != nil {
		l.Logger.Debugf(
			"found records for [%s] %q",
			dns.TypeToString[l.QType],
			l.QName,
		)
		return d, nil
	}

	dg, err := l.delegate(z)
	if err != nil {
		return nil, err
	}
	if dg != nil {
		l.Logger.Debugf("found delegate zone records for %q", l.QName)
		return dg, nil
	}

	l.Logger.Debugf(
		"no records found for [%s] %q",
		dns.TypeToString[l.QType],
		l.QName,
	)
	return &Response{Result: NameError}, nil
}

func (l *Lookup) settleViews() {
	for _, v := range l.ViewNames {
		if slices.Contains(l.ViewExclude, v) {
			continue
		}
		l.settledViews = append(l.settledViews, v)
	}

}

func (l *Lookup) direct(z *netbox.Zone) (*Response, error) {
	qt := []string{dns.TypeToString[l.QType]}
	if l.QType == dns.TypeA || l.QType == dns.TypeAAAA {
		qt = append(qt, "CNAME")
	}
	rq := &netbox.RecordQuery{
		FQDN: l.QName,
		Type: qt,
		Zone: z,
	}
	recs, err := rq.GetRecords(l.Client)
	if err != nil {
		return nil, err
	}
	if len(recs) <= 0 {
		return nil, nil
	}
	arrs := make([]dns.RR, 0, len(recs))
	for _, r := range recs {
		rr, err := r.ToRR()
		if err != nil {
			return nil, err
		}
		arrs = append(arrs, rr)
	}
	ers, err := l.processExtra(z, arrs, l.Family)
	if err != nil {
		return nil, err
	}
	errs := make([]dns.RR, 0, len(ers))
	for _, r := range ers {
		rr, err := r.ToRR()
		if err != nil {
			return nil, err
		}
		errs = append(errs, rr)
	}
	c := l.filterRRByType(arrs, dns.TypeCNAME)
	if l.QType == dns.TypeCNAME || len(c) > 0 {
		arrs = append(arrs, errs...)
		errs = nil
	}
	r := &Response{
		Answer: arrs,
		Extra:  errs,
	}
	return r, nil
}

func (l *Lookup) delegate(z *netbox.Zone) (*Response, error) {
	rq := &netbox.RecordQuery{
		FQDN: l.QName,
		Type: []string{"NS"},
		Zone: z,
	}
	recs, err := rq.GetRecords(l.Client)
	if err != nil {
		return nil, err
	}
	if len(recs) <= 0 {
		return nil, nil
	}
	var nsrrs []dns.RR
	for _, rec := range recs {
		rr, err := rec.ToRR()
		if err != nil {
			return nil, err
		}
		nsrrs = append(nsrrs, rr)
	}
	ers, err := l.processExtra(z, nsrrs, l.Family)
	if err != nil {
		return nil, err
	}
	var errs []dns.RR
	for _, r := range ers {
		rr, err := r.ToRR()
		if err != nil {
			return nil, err
		}
		errs = append(errs, rr)
	}
	r := &Response{
		Ns:     nsrrs,
		Extra:  errs,
		Result: Delegation,
	}
	return r, nil
}
