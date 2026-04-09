# netboxdns

[![Go Reference](https://pkg.go.dev/badge/github.com/doubleu-labs/coredns-netbox-plugin-dns.svg)](https://pkg.go.dev/github.com/doubleu-labs/coredns-netbox-plugin-dns)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=doubleu-labs_coredns-netbox-plugin-dns&metric=coverage)](https://sonarcloud.io/summary/overall?id=doubleu-labs_coredns-netbox-plugin-dns)
[![Go Report Card](https://goreportcard.com/badge/github.com/doubleu-labs/coredns-netbox-plugin-dns)](https://goreportcard.com/report/github.com/doubleu-labs/coredns-netbox-plugin-dns)

*netboxdns* - provides resolution using
[Netbox DNS Plugin (netbox-plugin-dns)](https://github.com/sys4/netbox-plugin-dns)

## Description

The *netboxdns* plugin provides resolution for zones configured using
[netbox-plugin-dns](https://github.com/sys4/netbox-plugin-dns).

**Depends on `netbox-plugin-dns` version `1.5.4` or greater.**

**Validated for `netbox` >= `v4.5.4` and `netbox-plugin-dns` >= `v1.5.4`.**

### Zone status filter

Only zones with NetBox status `active` are served. Zones with status
`parked`, `deprecated`, or `reserved` are returned by the NetBox API but
are deliberately excluded from every serving path of this plugin (lookup,
AXFR, IXFR poller). This matches the semantic intent of those statuses:
they describe records-of-record, not zones to publish on the wire.

To temporarily take a zone out of service without deleting it from
NetBox, switch its status from `active` to `parked` — the plugin will
stop serving it on the next poll cycle.

### Catalog zones (RFC 9432)

The plugin can serve [RFC 9432](https://www.rfc-editor.org/rfc/rfc9432.html)
catalog zones to make secondaries auto-discover the list of zones they
should transfer from this hidden primary. There is **no Corefile setting**
for this — catalog publication is opt-in via a NetBox-side convention:

> A zone is treated as a catalog **iff** its `status` is `parked` **and**
> its name starts with `cat.` (case-insensitive).

To turn it on, create a zone in NetBox (e.g. `cat.example.com`), set its
status to `parked`, fill in `soa_mname`, `soa_rname` and the nameservers
the same way you would for any normal zone, and put it in the same
`view` as the member zones it should advertise. The plugin will:

- list it via the catalog detection rule above,
- synthesise the catalog content from every `active` zone in the same
  view (one PTR per member, owner format `id-<netbox_zone_id>.zones.<catalog>.`,
  plus the mandatory `version.<catalog>. TXT "2"` marker),
- serve it via AXFR/IXFR through the same `transfer` plugin pipeline as
  any other zone.

The catalog SOA serial is **not** taken from NetBox; it is managed by an
internal counter that starts at the unix time of plugin boot and only
advances when the membership of the catalog actually changes (a member
zone is added to / removed from the view). Per-zone record edits do not
churn the catalog serial.

Per-view catalogs are supported automatically: just create one
`cat.<something>` zone in each view.

Make sure your Corefile server block and the `transfer` directive list
the catalog zone alongside the member zones, e.g.:

```
example.com cat.example.com:53 {
    netboxdns example.com cat.example.com {
        token TOKEN
        url   URL
        view  external
    }
    transfer example.com cat.example.com {
        to *
    }
}
```

The account that the API token is tied to will need the following permissions:

- `netbox_dns.view_zone`
- `netbox_dns.view_record`

## Syntax

Available configuration options:

```nginx
netboxdns [ZONES...] {
    token TOKEN
    url URL
    timeout DURATION
    fallthrough [ZONES...]
    tls CERT KEY CACERT
    view VIEWS...
    view_exclude VIEWS...
    poll_interval DURATION
    ixfr_history COUNT
}
```

- **ZONES**: A space-delimited list of zones that the plugin will answer for

- **`token TOKEN` (REQUIRED)**: The API token used to authenticate requests
to the Netbox instance

- **`url URL` (REQUIRED)**: The URL that Netbox is accessible at

- **`timeout DURATION`** (DEFAULT=`5s`): A duration to time-out requests to the
Netbox API

- **`fallthrough`**: If no record exists, send the request to the next plugin.
  - **(OPTIONAL) `ZONES...`**: A space-delimited list of zones that requests
  should be forwarded to the next plugin. If requests are not in the specified
  zones, an empty response is returned.

- **`tls`**: Used to authenticate to the Netbox instance if it is using HTTPS.
  - `0 arguments`: Creates a TLS configuration that uses system CA certificates
    to validate the connection to the Netbox instance. Use when Netbox is using
    a server certificate signed by a public CA. The server does not authenticate
    the client.

  - `1 argument`: Path to the CA PEM file. Creates a TLS configuration that uses
    the specified CA certificate to validate the connection to the Netbox
    instance. Use when Netbox is using a server certificate signed by a private
    CA. The server does not authenticate the client.

  - `2 arguments`: Paths to the client certificate and private key PEM files.
    Creates a TLS configuration that uses system CA certificates to validate the
    connection to the Netbox instance. Use when certificates are needed to
    authenticate to the Netbox instance (mTLS) (Netbox Cloud).

  - `3 arguments`: Paths to the client certificate, private key, and CA PEM
    files. Creates a TLS configuration that uses the specified CA certificate to
    validate the connection to the Netbox instance. Use when certificates are
    needed to authenticate to the Netbox instance (mTLS) and Netbox is using a
    server certificate signed by a private CA.

- **`view VIEWS...`**: Space-delimited list of NetBox DNS view names. Only zones
  belonging to these views will be served. When a single view is given, the
  server-side API filter is used (efficient). With multiple views, all zones are
  fetched and filtered client-side. Mutually exclusive with `view_exclude`.

- **`view_exclude VIEWS...`**: Space-delimited list of NetBox DNS view names to
  exclude. Zones from all other views will be served. Mutually exclusive with
  `view`. Use this when it is easier to name the views you do *not* want than
  those you do.

- **`poll_interval DURATION`** (DEFAULT=`300s`): Interval between background
  polls of the NetBox API. Each cycle fetches all zones and their records to
  build IXFR snapshots. Set to a lower value (e.g. `10s`) for faster zone
  convergence; higher values reduce NetBox API load.

- **`ixfr_history COUNT`** (DEFAULT=`16`): Maximum number of IXFR snapshots
  stored per zone. When a secondary requests an IXFR with a serial older than
  the oldest snapshot, the plugin falls back to a full AXFR. Set to `0` to
  disable the poller entirely (AXFR-only mode).

## Metrics

When the CoreDNS `prometheus` plugin is enabled, `netboxdns` exports the
following collectors (all under the `coredns_netboxdns_` prefix):

| Metric | Type | Labels | Description |
| --- | --- | --- | --- |
| `requests_total` | counter | `zone`, `rcode` | DNS requests handled by the plugin. |
| `request_duration_seconds` | histogram | `zone` | Per-request handling latency. |
| `netbox_requests_total` | counter | `endpoint`, `code` | HTTP requests to NetBox (`endpoint` is `zones`/`records`/…; `code` is `OK`/`Not Found`/… or `error`). |
| `netbox_request_duration_seconds` | histogram | `endpoint` | NetBox API round-trip latency. |
| `transfers_total` | counter | `zone`, `kind` | Outgoing zone transfers. `kind` ∈ `axfr`, `ixfr_delta`, `ixfr_noop`, `ixfr_fallback`, `catalog_axfr`, `catalog_ixfr_delta`, `catalog_ixfr_noop`. |
| `poll_cycles_total` | counter | `result` | Poller cycles (`success` / `error`). |
| `poll_duration_seconds` | histogram | — | Full poll-cycle duration. |
| `zone_serial` | gauge | `zone` | Last observed SOA serial of each zone (catalog zones included). |
| `cache_snapshots` | gauge | `zone` | Snapshots currently held in the IXFR ring buffer. |
| `catalog_members` | gauge | `catalog` | Member zones currently published in each catalog zone. |

## Architecture

```
NetBox + netbox-plugin-dns (source of truth)
    │
    │ REST API (poll every poll_interval + on-demand per query)
    ▼
CoreDNS + netboxdns plugin (hidden primary)
    │
    ├── DNS queries     ← recursive resolvers
    ├── AXFR / IXFR     → secondary DNS servers (BIND / Knot / NSD)
    ├── Catalog zone    → automatic zone discovery for secondaries
    └── /metrics        → Prometheus → Grafana
```

- Every DNS query triggers a NetBox API call for zone matching and record
  lookup. Use the CoreDNS `cache` plugin in the same server block to reduce
  API traffic.
- The poller runs in the background at `poll_interval`, fetching all zones and
  records to build IXFR snapshots. When a secondary requests an incremental
  transfer, the diff is computed from these snapshots.
- If NetBox is unavailable, queries fail with SERVFAIL. Mitigate with the
  `cache` plugin (serves stale entries) or a recursive resolver with
  serve-stale support.
- Zone transfers serve data from the snapshot cache when available, falling
  back to live API calls.

## Example: Hidden Primary with Zone Transfers

A production-ready Corefile for a hidden primary that serves zones from the
`external` NetBox view, supports AXFR/IXFR for secondaries, publishes a
catalog zone for automatic zone discovery, and exports Prometheus metrics:

```nginx
example.com cat.example.com:53 {
    netboxdns example.com cat.example.com {
        token   {$NETBOX_API_TOKEN}
        url     https://netbox.example.com
        view    external
        timeout 10s
        tls     /etc/ssl/certs/ca.pem
        poll_interval  60s
        ixfr_history   20
    }
    transfer example.com cat.example.com {
        to 10.0.0.50 10.0.0.51
    }
    cache 30
    prometheus :9153
    log
    errors
}
```

- **netboxdns** — resolves queries and provides zone data from NetBox
- **transfer** — serves AXFR/IXFR to the listed secondary IPs (use `to *`
  to allow any requestor)
- **cache** — caches responses for 30 seconds, reducing NetBox API load
- **prometheus** — exports metrics on `:9153/metrics`
- The catalog zone (`cat.example.com`) must be listed in both the server
  block keys and the `transfer` directive

## Building

Clone the [coredns](https://github.com/coredns/coredns) repository and change
into its directory.

```sh
git clone https://github.com/coredns/coredns.git
```

```sh
cd coredns
```

Fetch the plugin and add it to `coredns`'s `go.mod` file:

```sh
go get -u github.com/doubleu-labs/coredns-netbox-plugin-dns
```

Update `plugin.cfg` in the root of the directory. The `netboxdns` declaration
should be inserted after `cache` if you want responses from Netbox to be
cached.

> **Important:** if you intend to serve outgoing zone transfers (AXFR/IXFR)
> via the CoreDNS `transfer` plugin, `netboxdns` MUST be listed **after**
> `transfer:transfer` in `plugin.cfg`. Otherwise the AXFR query is handled
> by `netboxdns`'s normal `ServeDNS` lookup path (which forwards
> `?type=AXFR` to NetBox and gets a 400) instead of being routed to the
> plugin's `Transfer()` implementation.

```sh
# Using sed (insert after transfer so AXFR/IXFR works)
sed -i '/^transfer:transfer/a netboxdns:github.com/doubleu-labs/coredns-netbox-plugin-dns' plugin.cfg
```

```powershell
# Using Powershell
(Get-Content plugin.cfg).`
Replace("transfer:transfer", "transfer:transfer`nnetboxdns:github.com/doubleu-labs/coredns-netbox-plugin-dns") | `
Set-Content -Path plugin.cfg
```

### A note on the API authentication header

This plugin sends NetBox API requests with `Authorization: Bearer <token>`
rather than the legacy `Authorization: Token <token>` form. NetBox 4.x
accepts both, but `Bearer` is the form standardised by RFC 6750 and is
what newer NetBox-side tooling expects, so it is preferred going forward.
No configuration change is required on your side — just be aware when
auditing access logs or comparing against older examples that use
`Token <token>`.

> **Note:** The latest tagged release (`v0.2.1`) has a known bug with
> A/AAAA record resolution. Until a new release is tagged, build from the
> `main` branch using `go mod replace`:
>
> ```sh
> git clone https://github.com/doubleu-labs/coredns-netbox-plugin-dns.git /tmp/netboxdns
> cd coredns
> go mod edit -replace github.com/doubleu-labs/coredns-netbox-plugin-dns=/tmp/netboxdns
> ```

Build using `make`:

```sh
make
```

Or if `make` is not available, run:

```sh
go generate && go build
```

The `coredns` binary will be in the root of the project directory, unless
otherwise specified by the `-o` flag.

## Contributing

A [`justfile`](./justfile) is provided with commands to help with development.
It uses Podman and Podman Compose to run Netbox with the installed plugins in
a containerized environment with PostgreSQL and Valkey.

To start the test instance, run:

```sh
just instance-start
```

This will initialize Netbox and wait for it to become healthy, then populate the
database with the test dataset.

This can be browsed by visiting
[http://localhost:9999](http://localhost:9999/) with the `admin:admin` username
and password.

To stop the test instance, run:

```sh
just instance-stop
```

To run test and generate coverage the report, run:

```sh
just test
```

To view the coverage report using Go's built-in tool, run:

```sh
just coverage
```

`test` depends on `instance-start` and `coverage` depends on `test`, so they
will be run automatically when `just test` or `just coverage` is run. 
