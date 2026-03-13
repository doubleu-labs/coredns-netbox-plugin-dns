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
    tls CERT KET CACERT
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

```sh
# Using sed
sed -i '/^cache:cache/a netboxdns:github.com/doubleu-labs/coredns-netbox-plugin-dns' plugin.cfg
```

```powershell
# Using Powershell
(Get-Content plugin.cfg).`
Replace("cache:cache", "cache:cache`nnetboxdns:github.com/doubleu-labs/coredns-netbox-plugin-dns") | `
Set-Content -Path plugin.cfg
```

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
