package lookup

import (
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/miekg/dns"
)

// processOrigin returns nil unless the given name matches the zone's name AND
// the query type is SOA or NS. An NS request returns NS records and the
// associated A/AAAA records. An SOA request returns the SOA and NS records, in
// addition to any associated A/AAAA records.
func (l *Lookup) processOrigin(z *netbox.Zone, n string) (*Response, error) {
	if n != z.Name {
		return nil, nil
	}
	var qt []string
	switch l.QType {
	case dns.TypeSOA:
		qt = []string{"SOA", "NS"}
	case dns.TypeNS:
		qt = []string{"NS"}
	default:
		return nil, nil
	}
	rq := &netbox.RecordQuery{
		Name: "@",
		Type: qt,
		Zone: z,
	}
	recs, err := rq.GetRecords(l.Client)
	if err != nil {
		return nil, err
	}
	rrs := make([]dns.RR, 0, len(recs))
	for _, rec := range recs {
		var err error
		rr, err := rec.ToRR()
		if err != nil {
			return nil, err
		}
		rrs = append(rrs, rr)
	}
	a := l.filterRRByType(rrs, dns.TypeSOA)
	ns := l.filterRRByType(rrs, dns.TypeNS)
	e, err := l.processExtra(z, ns, l.Family)
	if err != nil {
		return nil, err
	}
	if len(e) == 0 {
		var err error
		e, err = l.processExtra(nil, ns, l.Family)
		if err != nil {
			return nil, err
		}
	}
	errs := make([]dns.RR, 0, len(e))
	for _, rec := range e {
		rr, err := rec.ToRR()
		if err != nil {
			return nil, err
		}
		errs = append(errs, rr)
	}
	if l.QType == dns.TypeNS {
		a = ns
		ns = nil
	}
	out := &Response{
		Answer: a,
		Ns:     ns,
		Extra:  errs,
	}
	return out, nil
}

// processExtra looks for associated records for the given dns.RR slice.
// It returns a slice of netbox.Record containing end resources referenced by
// CNAME, MX, NS, and SRV records.
func (l *Lookup) processExtra(
	z *netbox.Zone,
	rrs []dns.RR,
	f int,
) ([]netbox.Record, error) {
	var out []netbox.Record
	for _, rr := range rrs {
		var n string
		switch t := rr.(type) {
		case *dns.CNAME:
			n = t.Target
		case *dns.MX:
			n = t.Mx
		case *dns.NS:
			n = t.Ns
		case *dns.SRV:
			n = t.Target
		}
		if len(n) == 0 {
			continue
		}
		var t []string
		switch f {
		case 1:
			t = []string{"A"}
		case 2:
			t = []string{"AAAA"}
		}
		rq := &netbox.RecordQuery{
			FQDN: n,
			Type: t,
			Zone: z,
		}
		rs, err := rq.GetRecords(l.Client)
		if err != nil {
			return nil, err
		}
		out = append(out, rs...)
	}
	return out, nil
}

// filterRRByType returns a slice of dns.RR that match the given type.
func (l *Lookup) filterRRByType(rrs []dns.RR, rt uint16) []dns.RR {
	var out []dns.RR
	for _, rr := range rrs {
		if rr.Header().Rrtype == rt {
			out = append(out, rr)
		}
	}
	return out
}
