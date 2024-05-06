package netboxdns

import (
	"strings"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/miekg/dns"
)

type lookupResult int

const (
	lookupSuccess    lookupResult = iota
	lookupNameError               // NXDomain
	lookupDelegation              // Delegate, non-authoritative
)

type lookupResponse struct {
	Answer       []dns.RR
	Ns           []dns.RR
	Extra        []dns.RR
	LookupResult lookupResult
}

func (netboxdns *NetboxDNS) lookup(
	name string,
	qtype uint16,
	family int,
) (*lookupResponse, error) {
	nameTrimmed := strings.TrimSuffix(name, ".")
	// check if zone exists on Netbox
	zone, err := netboxdns.matchZone(nameTrimmed)
	if err != nil {
		return nil, err
	}
	if zone == nil {
		return &lookupResponse{LookupResult: lookupNameError}, nil
	}

	// check if qname is for zone origin
	if nameTrimmed == zone.Name {
		originResponse, err := netboxdns.processOrigin(qtype, zone, family)
		if err != nil {
			return nil, err
		}
		if originResponse != nil {
			return originResponse, nil
		}
	}

	// lookup exact request
	direct, err := netboxdns.lookupDirect(nameTrimmed, qtype, zone, family)
	if err != nil {
		return nil, err
	}
	if direct != nil {
		return direct, nil
	}

	// if no exact records exist for the request, check if the qname is a
	// delegate zone
	delegate, err := netboxdns.lookupDelegate(nameTrimmed, zone, family)
	if err != nil {
		return nil, err
	}
	if delegate != nil {
		return delegate, nil
	}

	return &lookupResponse{LookupResult: lookupNameError}, nil
}

func (netboxdns *NetboxDNS) matchZone(qname string) (*netbox.Zone, error) {
	managedZones, err := netbox.GetZones(netboxdns.requestClient)
	if err != nil {
		return nil, err
	}
	var out *netbox.Zone
	for _, managedZone := range managedZones {
		if dns.IsSubDomain(managedZone.Name, qname) {
			if out == nil {
				out = &managedZone
			}
			if len(managedZone.Name) > len(out.Name) {
				out = &managedZone
			}
		}
	}
	return out, nil
}

func (netboxdns *NetboxDNS) processOrigin(
	qtype uint16,
	zone *netbox.Zone,
	family int,
) (*lookupResponse, error) {
	var queryType []string
	switch qtype {
	case dns.TypeSOA:
		queryType = []string{"SOA", "NS"}
	case dns.TypeNS:
		queryType = []string{"NS"}
	default:
		return nil, nil
	}
	records, err := netbox.GetRecordsQuery(
		netboxdns.requestClient,
		&netbox.RecordQuery{
			Name: "@",
			Type: queryType,
			Zone: zone,
		},
	)
	if err != nil {
		return nil, err
	}
	rrs, err := recordsToRR(records)
	if err != nil {
		return nil, err
	}
	answer := filterRRByType(rrs, dns.TypeSOA)
	ns := filterRRByType(rrs, dns.TypeNS)
	extraRecords, err := netboxdns.processExtra(ns, zone, family)
	if err != nil {
		return nil, err
	}
	if len(extraRecords) == 0 {
		// if no A/AAAA records exist for the NS in the specified zone, check if
		// the server has records anywhere
		extraRecords, err = netboxdns.processExtra(ns, nil, family)
		if err != nil {
			return nil, err
		}
	}
	extra, err := recordsToRR(extraRecords)
	if err != nil {
		return nil, err
	}
	if qtype == dns.TypeNS {
		answer = ns
		ns = nil
	}
	return &lookupResponse{
		Answer: answer,
		Ns:     ns,
		Extra:  extra,
	}, nil
}

func (netboxdns *NetboxDNS) processExtra(
	answer []dns.RR,
	zone *netbox.Zone,
	family int,
) ([]netbox.Record, error) {
	var out []netbox.Record
	for _, rr := range answer {
		name := ""
		switch t := rr.(type) {
		case *dns.SRV:
			name = t.Target
		case *dns.MX:
			name = t.Mx
		case *dns.NS:
			name = t.Ns
		case *dns.CNAME:
			name = t.Target
		}
		if len(name) == 0 {
			continue
		}
		var reqType []string
		switch family {
		case 1:
			reqType = []string{"A"}
		case 2:
			reqType = []string{"AAAA"}
		}
		records, err := netbox.GetRecordsQuery(
			netboxdns.requestClient,
			&netbox.RecordQuery{
				FQDN: strings.TrimSuffix(name, "."),
				Type: reqType,
				Zone: zone,
			},
		)
		if err != nil {
			return out, err
		}
		out = append(out, records...)
	}
	return out, nil
}

func (netboxdns *NetboxDNS) lookupDirect(
	qname string,
	qtype uint16,
	zone *netbox.Zone,
	family int,
) (*lookupResponse, error) {
	records, err := netbox.GetRecordsQuery(
		netboxdns.requestClient,
		&netbox.RecordQuery{
			FQDN: qname,
			Type: []string{dns.TypeToString[qtype]},
			Zone: zone,
		},
	)
	if err != nil {
		return nil, err
	}

	if len(records) > 0 {
		answer, err := recordsToRR(records)
		if err != nil {
			return nil, err
		}
		extraRecords, err := netboxdns.processExtra(answer, zone, family)
		if err != nil {
			return nil, err
		}
		extra, err := recordsToRR(extraRecords)
		if err != nil {
			return nil, err
		}
		if qtype == dns.TypeCNAME {
			answer = append(answer, extra...)
			extra = nil
		}
		return &lookupResponse{
			Answer: answer,
			Extra:  extra,
		}, nil
	}
	return nil, nil
}

func (netboxdns *NetboxDNS) lookupDelegate(
	qname string,
	zone *netbox.Zone,
	family int,
) (*lookupResponse, error) {
	records, err := netbox.GetRecordsQuery(
		netboxdns.requestClient,
		&netbox.RecordQuery{
			FQDN: qname,
			Type: []string{"NS"},
			Zone: zone,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(records) > 0 {
		ns, err := recordsToRR(records)
		if err != nil {
			return nil, err
		}
		extraRecords, err := netboxdns.processExtra(ns, nil, family)
		if err != nil {
			return nil, err
		}
		extra, err := recordsToRR(extraRecords)
		if err != nil {
			return nil, err
		}
		return &lookupResponse{
			Ns:           ns,
			Extra:        extra,
			LookupResult: lookupDelegation,
		}, nil
	}
	return nil, nil
}
