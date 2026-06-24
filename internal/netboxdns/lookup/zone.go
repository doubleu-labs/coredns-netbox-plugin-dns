package lookup

import (
	"strings"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/miekg/dns"
)

// matchZone returns the zone that matches the given name. If the name exists in
// multiple views, the last checked view is returned and a warning is logged.
func (l *Lookup) matchZone(n string) (*netbox.Zone, error) {
	zs, err := netbox.GetZones(l.Client, l.settledViews)
	if err != nil {
		return nil, err
	}
	var out *netbox.Zone
	var amb int
	for i := range zs {
		z := &zs[i]
		if dns.IsSubDomain(z.Name, n) {
			if out == nil {
				out = z
			}
			if len(z.Name) > len(out.Name) {
				out = z
			}
			if strings.EqualFold(z.Name, n) {
				amb++
			}
		}
	}
	if amb > 1 && len(l.settledViews) == 0 {
		l.Logger.Warningf(
			"zone %q exists in %d views; configure 'view' or 'view_exclude' "+
				"to disambiguate",
			n,
			amb,
		)
	}
	return out, nil
}

func (l *Lookup) catalogZones(n string) (*Response, error) {
	if l.QType != dns.TypeSOA && l.QType != dns.TypeNS {
		return nil, nil
	}

	zs, err := netbox.GetCatalogZones(l.Client, l.settledViews, l.CatalogPrefix)
	if err != nil {
		return nil, err
	}
	var cz *netbox.Zone
	for k, v := range zs {
		if strings.EqualFold(v.Name, n) {
			cz = &zs[k]
			break
		}
	}
	if cz == nil {
		return nil, nil
	}

	rq := &netbox.RecordQuery{
		Zone: cz,
	}
	switch l.QType {
	case dns.TypeSOA:
		rq.Type = []string{"SOA", "NS"}
	case dns.TypeNS:
		rq.Type = []string{"NS"}
	}
	recs, err := rq.GetRecords(l.Client)
	if err != nil {
		return nil, err
	}
	crrs := make([]dns.RR, 0, len(recs))
	for _, rec := range recs {
		var rrErr error
		rr, rrErr := rec.ToRR()
		if rrErr != nil {
			return nil, err
		}
		crrs = append(crrs, rr)
	}
	r := &Response{
		Result: Success,
	}
	switch l.QType {
	case dns.TypeSOA:
		r.Answer = l.filterRRByType(crrs, dns.TypeSOA)
		r.Ns = l.filterRRByType(crrs, dns.TypeNS)
	case dns.TypeNS:
		r.Answer = l.filterRRByType(crrs, dns.TypeNS)
	}

	return r, nil
}
