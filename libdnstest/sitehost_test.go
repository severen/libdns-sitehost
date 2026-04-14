package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"

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
	suite.RunTests(t)
}
