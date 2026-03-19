# SiteHost Provider for libdns

This Go package implements a provider for
[libdns](https://github.com/libdns/libdns) to manage DNS records which are
hosted by [SiteHost](https://sitehost.nz/).

Most notably, this provider can be used by Caddy (or certmagic) for managing DNS
records as part of answering an [ACME DNS-01
challenge](https://letsencrypt.org/docs/challenge-types/#dns-01-challenge), such
as implemented by [our Caddy
module](https://github.com/sitehostnz/caddy-sitehost).

## Configuration

This provider is configured via the following struct fields:

| Field      | JSON Key    | Required | Default             | Description           |
|------------|-------------|----------|---------------------|-----------------------|
| `ClientID` | `client_id` | Yes      | N/A                 | SiteHost client ID    |
| `APIKey`   | `api_key`   | Yes      | N/A                 | SiteHost API key      |
| `APIHost`  | `api_host`  | No       | `"api.sitehost.nz"` | SiteHost API hostname |

The client ID should be the numeric ID of a client that has access to the DNS
zones you wish to manage. In most cases, this is the same client that owns the
DNS zones, but it may also be a parent of a sub-account. For more information on
the SiteHost API and creating a key, [refer to our knowledge base
article](https://kb.sitehost.nz/developers/api).

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"os"
	"time"

	"github.com/libdns/libdns"
	sitehost "github.com/sitehostnz/libdns-sitehost"
)

func main() {
	provider := &sitehost.Provider{
		ClientID: os.Getenv("SITEHOST_CLIENT_ID"),
		APIKey:   os.Getenv("SITEHOST_API_KEY"),
	}

	ip, err := netip.ParseAddr("192.168.0.1")
	if err != nil {
		log.Fatalf("Error parsing IP address: %v", err)
	}

	records := []libdns.Record{
		libdns.Address{
			Name: "@",
			IP:   ip,
		},
		libdns.TXT{
			Name: "hello",
			Text: "Hello, world!",
			TTL:  300 * time.Second,
		},
	}

	ctx := context.Background()

	// Create the records.
	createdRecords, err := provider.AppendRecords(ctx, "example.co.nz", records)
	if err != nil {
		log.Fatalf("Error creating records: %v", err)
	}

	for _, record := range createdRecords {
		rr := record.RR()
		fmt.Printf("Created %s record: %s -> %s\n", rr.Type, rr.Name, rr.Data)
	}

	// Delete the records.
	deletedRecords, err := provider.DeleteRecords(ctx, "example.co.nz", createdRecords)
	if err != nil {
		log.Fatalf("Error deleting records: %v", err)
	}

	for _, record := range deletedRecords {
		rr := record.RR()
		fmt.Printf("Deleted %s record: %s\n", rr.Type, rr.Name)
	}
}
```

For more, see the [libdns
documentation](https://pkg.go.dev/github.com/libdns/libdns).

## Caveats

There are a couple caveats one should be aware of when using this library:

- We only support setting the TTL for a whole zone, not individual records.
  Consequently, any TTL values for records are ignored by `AddRecords` and
  `SetRecords`.
- Operations on several records at once are not guaranteed to be atomic. This
  means that if an error occurs partway through, the zone may be left in a
  partially-updated state and require correction.
