package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/libdns/libdns"
	"github.com/libdns/libdns/libdnstest"
	sitehost "github.com/sitehostnz/libdns-sitehost"
)

func TestMain(m *testing.M) {
	// Load .env file if present, without overriding existing environment
	// variables.
	if f, err := os.Open(".env"); err == nil {
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			if key, val, ok := strings.Cut(line, "="); ok {
				if _, exists := os.LookupEnv(key); !exists {
					if err := os.Setenv(key, val); err != nil {
						panic(fmt.Sprintf("failed to set env var %s: %v", key, err))
					}
				}
			}
		}
	}

	os.Exit(m.Run())
}

func TestSiteHostProvider(t *testing.T) {
	clientID := os.Getenv("SITEHOST_CLIENT_ID")
	apiKey := os.Getenv("SITEHOST_API_KEY")
	testZone := os.Getenv("SITEHOST_TEST_ZONE")
	apiHost := os.Getenv("SITEHOST_API_HOST")

	if clientID == "" || apiKey == "" || testZone == "" {
		t.Fatal("SITEHOST_CLIENT_ID, SITEHOST_API_KEY, and SITEHOST_TEST_ZONE must be set (via environment or .env file)")
	}

	if !strings.HasSuffix(testZone, ".") {
		t.Fatal("SITEHOST_TEST_ZONE must have a trailing dot (e.g. 'example.com.')")
	}

	provider := &sitehost.Provider{
		ClientID: clientID,
		APIKey:   apiKey,
		APIHost:  apiHost,
	}

	suite := libdnstest.NewTestSuite(provider, testZone)
	// We currently do not support SVCB and HTTPS record types at SiteHost.
	suite.SkipRRTypes = map[string]bool{
		"SVCB":  true,
		"HTTPS": true,
	}

	// SiteHost only supports a TTL for a whole zone, so records come back with
	// the zone's TTL. The suite compares records exactly, so give its records
	// that TTL to make them compare equal.
	ttl := zoneTTL(t, provider, testZone)
	suite.AppendRecordFunc = func(record libdns.Record) libdns.Record {
		rr := record.RR()
		rr.TTL = ttl
		parsed, err := rr.Parse()
		if err != nil {
			t.Fatalf("setting TTL on test record %s %s: %v", rr.Name, rr.Type, err)
		}
		return parsed
	}

	// The cleanup in libdns v1.2.0-alpha.1 only matches names starting with
	// "test-", so it misses SRV test records (fixed upstream after that tag).
	// Until the next libdns release, leftover SRV test records need deleting
	// by hand.
	suite.RunTests(t)
}

// zoneTTL returns the TTL of the zone, taken from its root NS records, which
// every SiteHost zone has and which cannot be removed.
func zoneTTL(t *testing.T, provider *sitehost.Provider, zone string) time.Duration {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	records, err := provider.GetRecords(ctx, zone)
	if err != nil {
		t.Fatalf("getting records to find the zone's TTL: %v", err)
	}

	for _, record := range records {
		rr := record.RR()
		if rr.Name == "@" && rr.Type == "NS" {
			return rr.TTL
		}
	}

	t.Fatalf("no root NS record found in zone %s to take its TTL from", zone)
	return 0
}
