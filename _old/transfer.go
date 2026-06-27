package netboxdns

import (
	"strings"

	"github.com/coredns/coredns/plugin/transfer"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/zonecache"
	"github.com/miekg/dns"
)

// transferBatchSize controls how many RRs are written to the transfer channel
// per send. The CoreDNS transfer plugin packs each batch into a single DNS
// message, so values around 100 strike a reasonable balance between message
// size and goroutine wake-ups for large zones.
const transferBatchSize = 100

// Transfer implements the transfer.Transferer interface so the CoreDNS
// `transfer` plugin can serve outgoing zone transfers (AXFR/IXFR) for zones
// managed by NetBox.
//
// Contract (verbatim from coredns/plugin/transfer):
//   - serial == 0  → AXFR: send the full zone with SOA first and last.
//   - serial >= current → IXFR no-op: send a single SOA and close the channel.
//   - serial <  current → IXFR: AXFR fallback (this phase has no diff engine yet).
//
// The implementation always synthesizes the SOA from NetBox zone metadata
// rather than from the SOA Record returned by the records endpoint, which
// keeps the wire form deterministic and avoids depending on miekg/dns being
// able to round-trip the netbox-formatted SOA value string.
func (n *NetboxDNS) Transfer(zone string, serial uint32) (
	<-chan []dns.RR,
	error,
) {
	zoneName := strings.TrimSuffix(zone, ".")
	nbZone, err := n.findZone(zoneName)
	if err != nil {
		return nil, err
	}
	if nbZone == nil {
		return nil, transfer.ErrNotAuthoritative
	}

	// Catalog zones go through their own transfer path: their content is
	// synthesised from active member zones rather than fetched from
	// /records/, and their SOA serial comes from catalogTracker (which is
	// reflected in the cache snapshot the poller wrote).
	if isCatalogZone(nbZone) {
		return n.transferCatalog(nbZone, serial)
	}

	soa := buildSOA(nbZone, dns.Fqdn(nbZone.Name))

	// IXFR no-op: requester already has the current (or newer) serial.
	if serial != 0 && serial >= nbZone.SOASerial {
		n.metrics.TransfersTotal.Inc(nbZone.Name, "ixfr_noop")
		ch := make(chan []dns.RR, 1)
		ch <- []dns.RR{soa}
		close(ch)
		return ch, nil
	}

	// IXFR delta: try to satisfy from the snapshot cache. The cache is
	// only populated when the poller is enabled (ixfr_history > 0).
	if serial != 0 && n.cache != nil {
		if from, to, ok := n.cache.Diff(
			nbZone.Name,
			serial,
		); ok && to.Serial == nbZone.SOASerial {
			oldSOA := buildSOAWithSerial(
				nbZone,
				dns.Fqdn(nbZone.Name),
				from.Serial,
			)
			n.metrics.TransfersTotal.Inc(nbZone.Name, "ixfr_delta")
			return n.streamIXFR(soa, oldSOA, from, to), nil
		}
	}

	// AXFR (serial==0) or IXFR fallback when the cache cannot serve a
	// delta. The kind label distinguishes the two so AXFR fallbacks are
	// visible without an extra metric.
	if serial == 0 {
		n.metrics.TransfersTotal.Inc(nbZone.Name, "axfr")
	} else {
		n.metrics.TransfersTotal.Inc(nbZone.Name, "ixfr_fallback")
	}

	ch := make(chan []dns.RR)
	go n.streamZoneData(ch, nbZone, zone, soa)

	return ch, nil
}

func (n *NetboxDNS) streamZoneData(
	ch chan<- []dns.RR,
	nbZone *netbox.Zone,
	zone string,
	soa *dns.SOA,
) {
	defer close(ch)

	records, err := netbox.GetRecordsQuery(
		n.requestClient,
		&netbox.RecordQuery{Zone: nbZone},
	)
	if err != nil {
		logger.Errorf("transfer %s: fetching records: %v", zone, err)
		return
	}

	// Drop SOA records returned by NetBox; we send the synthesized one
	// as the opening and closing RR instead.
	filtered := make([]netbox.Record, 0, len(records))
	for i := range records {
		if records[i].Type == "SOA" {
			continue
		}
		filtered = append(filtered, records[i])
	}

	rrs, err := recordsToRR(filtered)
	if err != nil {
		logger.Errorf("transfer %s: converting records: %v", zone, err)
		return
	}

	// Opening SOA.
	ch <- []dns.RR{soa}

	sendBatched(ch, rrs)

	// Closing SOA (RFC 5936 §2.2 — "the last RR sent in the answer
	// section MUST also be the SOA").
	ch <- []dns.RR{soa}
}

// transferCatalog serves AXFR/IXFR for a catalog zone. The catalog body
// is taken from the snapshot cache when available; on a cold start (cache
// disabled or never populated for this catalog) we build it inline by
// fetching the active member zones once. The IXFR delta path is the same
// streamIXFR helper used by member zones — diff/cache code does not need
// to know that this is a catalog.
func (n *NetboxDNS) transferCatalog(
	c *netbox.Zone,
	serial uint32,
) (<-chan []dns.RR, error) {
	var (
		latest    catalog_old.Snapshot
		hasCached bool
	)
	if n.cache != nil {
		latest, hasCached = n.cache.Latest(c.Name)
	}
	if !hasCached {
		members, err := n.getActiveZones()
		if err != nil {
			return nil, err
		}
		latest = catalog_old.Snapshot{
			Serial: n.catalogTracker.NextSerial(c.Name, members),
			RRs:    buildCatalog(c, members),
		}
		if n.cache != nil {
			n.cache.Put(c.Name, latest.Serial, latest.RRs)
		}
	}

	soa := buildSOAWithSerial(c, dns.Fqdn(c.Name), latest.Serial)

	// IXFR no-op.
	if serial != 0 && serial >= latest.Serial {
		n.metrics.TransfersTotal.Inc(c.Name, "catalog_ixfr_noop")
		ch := make(chan []dns.RR, 1)
		ch <- []dns.RR{soa}
		close(ch)
		return ch, nil
	}

	// IXFR delta from cache.
	if serial != 0 && n.cache != nil {
		if from, to, ok := n.cache.Diff(
			c.Name,
			serial,
		); ok && to.Serial == latest.Serial {
			oldSOA := buildSOAWithSerial(
				c,
				dns.Fqdn(c.Name),
				from.Serial,
			)
			n.metrics.TransfersTotal.Inc(c.Name, "catalog_ixfr_delta")
			return n.streamIXFR(soa, oldSOA, from, to), nil
		}
	}

	// AXFR / IXFR fallback.
	n.metrics.TransfersTotal.Inc(c.Name, "catalog_axfr")
	ch := make(chan []dns.RR)
	go func() {
		defer close(ch)
		ch <- []dns.RR{soa}
		sendBatched(ch, latest.RRs)
		ch <- []dns.RR{soa}
	}()
	return ch, nil
}

// streamIXFR returns an RFC 1995 §4 incremental transfer message:
//
//	newSOA
//	oldSOA  <removed RRs...>  newSOA  <added RRs...>
//	newSOA
//
// Removed and added blocks are computed by zonecache.Diff. The body is sent
// in the same fixed-size batches as AXFR so the wire framing is consistent.
func (n *NetboxDNS) streamIXFR(
	newSOA *dns.SOA,
	oldSOA *dns.SOA,
	from catalog_old.Snapshot,
	to catalog_old.Snapshot,
) <-chan []dns.RR {
	removed, added := zonecache.Diff(from.RRs, to.RRs)

	ch := make(chan []dns.RR)
	go func() {
		defer close(ch)
		ch <- []dns.RR{newSOA}
		ch <- []dns.RR{oldSOA}
		sendBatched(ch, removed)
		ch <- []dns.RR{newSOA}
		sendBatched(ch, added)
		ch <- []dns.RR{newSOA}
	}()
	return ch
}

// sendBatched chunks rrs into the transfer channel. Empty input is a no-op
// (an empty removed or added section is valid in IXFR).
func sendBatched(ch chan<- []dns.RR, rrs []dns.RR) {
	for start := 0; start < len(rrs); start += transferBatchSize {
		end := start + transferBatchSize
		if end > len(rrs) {
			end = len(rrs)
		}
		ch <- rrs[start:end]
	}
}

// buildSOAWithSerial returns a *dns.SOA identical to buildSOA(zone, fqdn)
// but with the Serial field overridden. Used to construct the "old" SOA
// that opens each IXFR difference sequence (RFC 1995 §4).
func buildSOAWithSerial(
	zone *netbox.Zone,
	fqdn string,
	serial uint32,
) *dns.SOA {
	soa := buildSOA(zone, fqdn)
	soa.Serial = serial
	return soa
}

// buildSOA constructs a *dns.SOA from a NetBox Zone's SOA fields.
func buildSOA(zone *netbox.Zone, fqdn string) *dns.SOA {
	return &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   fqdn,
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    zone.SOATTL,
		},
		Ns:      dns.Fqdn(zone.SOAMName.Name),
		Mbox:    dns.Fqdn(zone.SOARName),
		Serial:  zone.SOASerial,
		Refresh: zone.SOARefresh,
		Retry:   zone.SOARetry,
		Expire:  zone.SOAExpire,
		Minttl:  zone.SOAMinimum,
	}
}

// findZone returns the (single) zone whose Name matches name (case
// insensitive). It looks first at active member zones and then at catalog
// zones (parked + cat.* prefix). The lookup honours the configured view
// filter so a hidden primary only sees the zones it is meant to serve.
// Returns (nil, nil) when no zone matches.
func (n *NetboxDNS) findZone(name string) (*netbox.Zone, error) {
	zones, err := n.getActiveZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if strings.EqualFold(zones[i].Name, name) {
			return &zones[i], nil
		}
	}
	catalogs, err := n.getCatalogZones()
	if err != nil {
		return nil, err
	}
	for i := range catalogs {
		if strings.EqualFold(catalogs[i].Name, name) {
			return &catalogs[i], nil
		}
	}
	return nil, nil
}

// isCatalogZone reports whether a NetBox zone is a catalog zone declared
// via the "parked status + cat. name prefix" convention.
func isCatalogZone(z *netbox.Zone) bool {
	return z.Status == "parked" &&
		strings.HasPrefix(strings.ToLower(z.Name), "cat.")
}
