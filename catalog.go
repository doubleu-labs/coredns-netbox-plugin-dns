package netboxdns

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/miekg/dns"
)

// catalogTracker assigns SOA serials to catalog zones using a monotonic
// counter that only bumps when the membership of the catalog actually
// changes (option C from the design discussion). The first serial seen
// for a catalog is the unix epoch at boot, so two coredns instances
// restarted within the same second still produce strictly increasing
// serials over their lifetime as soon as anything in NetBox changes.
//
// Membership is fingerprinted by the sorted list of NetBox zone IDs of
// the active member zones. Renames, SOA tweaks, or per-zone record
// changes do NOT bump the catalog serial — only adding or removing a
// zone does, which matches RFC 9432 §4.2 (the catalog publishes the
// list of zones, not their contents).
type catalogTracker struct {
	mu        sync.Mutex
	serials   map[string]uint32
	prevPrint map[string]string
}

func newCatalogTracker() *catalogTracker {
	return &catalogTracker{
		serials:   make(map[string]uint32),
		prevPrint: make(map[string]string),
	}
}

// NextSerial returns the SOA serial that should be used for the next
// snapshot of catalog. If the membership fingerprint is unchanged from
// the previous call, the previous serial is returned (cache.Put is
// idempotent on equal serials). Otherwise the counter advances by one
// — or is initialised to time.Now().Unix() the very first time we see
// this catalog name.
func (ct *catalogTracker) NextSerial(catalog string, members []netbox.Zone) uint32 {
	print := membershipFingerprint(members)
	ct.mu.Lock()
	defer ct.mu.Unlock()
	prev, ok := ct.serials[catalog]
	if !ok {
		ct.serials[catalog] = uint32(time.Now().Unix())
		ct.prevPrint[catalog] = print
		return ct.serials[catalog]
	}
	if ct.prevPrint[catalog] == print {
		return prev
	}
	ct.serials[catalog] = prev + 1
	ct.prevPrint[catalog] = print
	return ct.serials[catalog]
}

// membershipFingerprint returns a stable string identifying the *set* of
// member zones. Only the zone IDs are considered (renames/SOA tweaks do
// not change membership).
func membershipFingerprint(members []netbox.Zone) string {
	ids := make([]int, 0, len(members))
	for i := range members {
		ids = append(ids, members[i].ID)
	}
	sort.Ints(ids)
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconvI(id)
	}
	return strings.Join(parts, ",")
}

func strconvI(n int) string { return fmt.Sprintf("%d", n) }

// serveCatalogMeta handles SOA and NS queries for catalog zones in the
// ServeDNS path. Catalog zones are status=parked, so the normal lookup
// code path (which uses GetZones → status=active) would never find them.
// Returns (nil, nil) when qname does not match any catalog zone, so the
// caller falls through to the regular lookup path.
func (n *NetboxDNS) serveCatalogMeta(qname string, qtype uint16) (*lookupResponse, error) {
	nameTrimmed := strings.TrimSuffix(qname, ".")
	catalogs, err := n.getCatalogZones()
	if err != nil {
		return nil, err
	}
	var catalog *netbox.Zone
	for i := range catalogs {
		if strings.EqualFold(catalogs[i].Name, nameTrimmed) {
			catalog = &catalogs[i]
			break
		}
	}
	if catalog == nil {
		return nil, nil
	}

	fqdn := dns.Fqdn(catalog.Name)

	// Determine the serial: prefer the cache (catalogTracker-assigned),
	// fall back to fetching members and computing inline.
	serial := catalog.SOASerial
	if n.cache != nil {
		if snap, ok := n.cache.Latest(catalog.Name); ok {
			serial = snap.Serial
		}
	}

	soa := buildSOAWithSerial(catalog, fqdn, serial)

	var ns []dns.RR
	for _, nsEntry := range catalog.NameServers {
		ns = append(ns, &dns.NS{
			Hdr: dns.RR_Header{
				Name:   fqdn,
				Rrtype: dns.TypeNS,
				Class:  dns.ClassINET,
				Ttl:    catalog.SOATTL,
			},
			Ns: dns.Fqdn(nsEntry.Name),
		})
	}

	switch qtype {
	case dns.TypeSOA:
		return &lookupResponse{
			Answer:       []dns.RR{soa},
			Ns:           ns,
			LookupResult: lookupSuccess,
		}, nil
	case dns.TypeNS:
		return &lookupResponse{
			Answer:       ns,
			LookupResult: lookupSuccess,
		}, nil
	}
	return nil, nil
}

// catalogVersion is the only catalog zone schema version defined by
// RFC 9432 §4.1. It MUST appear as 'version.<catalog>. TXT "2"'.
const catalogVersion = "2"

// buildCatalog returns the records that make up an RFC 9432 catalog zone.
//
// It does NOT include the synthesised opening/closing SOA — Transfer()
// builds that separately, the same way it does for normal zones, so the
// IXFR delta path stays uniform. The returned slice contains:
//
//   - one NS record per nameserver listed on the catalog zone in NetBox
//   - the mandatory 'version.<catalog>. TXT "2"' marker
//   - one PTR per member zone, owner '<id>.zones.<catalog>.' (RFC 9432 §4.2)
//
// memberZones must be the active zones in the same view as the catalog
// (callers get this from netbox.GetZones). The catalog zone itself is
// excluded from the PTR list defensively, even though active+parked
// statuses normally guarantee it can't appear there.
func buildCatalog(catalog *netbox.Zone, memberZones []netbox.Zone) []dns.RR {
	cName := dns.Fqdn(catalog.Name)
	out := make([]dns.RR, 0, 2+len(memberZones))

	// Apex NS records (one per nameserver configured on the catalog zone).
	for _, ns := range catalog.NameServers {
		out = append(out, &dns.NS{
			Hdr: dns.RR_Header{
				Name:   cName,
				Rrtype: dns.TypeNS,
				Class:  dns.ClassINET,
				Ttl:    catalog.SOATTL,
			},
			Ns: dns.Fqdn(ns.Name),
		})
	}

	// Mandatory version marker.
	out = append(out, &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   "version." + cName,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    catalog.SOATTL,
		},
		Txt: []string{catalogVersion},
	})

	// One PTR per member zone. The unique-id label must be stable across
	// poll cycles so secondaries can identify zones across snapshots; we
	// use the NetBox zone ID, which never changes.
	cLower := strings.ToLower(strings.TrimSuffix(catalog.Name, "."))
	for i := range memberZones {
		m := &memberZones[i]
		if strings.EqualFold(m.Name, cLower) {
			continue
		}
		owner := fmt.Sprintf("id-%d.zones.%s", m.ID, cName)
		out = append(out, &dns.PTR{
			Hdr: dns.RR_Header{
				Name:   owner,
				Rrtype: dns.TypePTR,
				Class:  dns.ClassINET,
				Ttl:    catalog.SOATTL,
			},
			Ptr: dns.Fqdn(m.Name),
		})
	}

	return out
}
