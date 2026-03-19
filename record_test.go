package sitehost

import (
	"net/netip"
	"testing"
	"time"

	"github.com/libdns/libdns"
	"github.com/sitehostnz/gosh/pkg/models"
)

func TestRecordID(t *testing.T) {
	tests := []struct {
		name   string
		record libdns.Record
		want   string
	}{
		{
			name:   "Address with provider data",
			record: libdns.Address{IP: netip.MustParseAddr("1.2.3.4"), ProviderData: providerData{ID: "123"}},
			want:   "123",
		},
		{
			name:   "Address without provider data",
			record: libdns.Address{IP: netip.MustParseAddr("1.2.3.4")},
			want:   "",
		},
		{
			name:   "CNAME with provider data",
			record: libdns.CNAME{Target: "example.com.", ProviderData: providerData{ID: "456"}},
			want:   "456",
		},
		{
			name:   "MX with provider data",
			record: libdns.MX{Target: "mail.example.com.", ProviderData: providerData{ID: "789"}},
			want:   "789",
		},
		{
			name:   "NS with provider data",
			record: libdns.NS{Target: "ns1.example.com.", ProviderData: providerData{ID: "101"}},
			want:   "101",
		},
		{
			name:   "TXT with provider data",
			record: libdns.TXT{Text: "v=spf1", ProviderData: providerData{ID: "202"}},
			want:   "202",
		},
		{
			name:   "SRV with provider data",
			record: libdns.SRV{Target: "sip.example.com.", ProviderData: providerData{ID: "303"}},
			want:   "303",
		},
		{
			name:   "CAA with provider data",
			record: libdns.CAA{Tag: "issue", Value: "letsencrypt.org", ProviderData: providerData{ID: "404"}},
			want:   "404",
		},
		{
			name:   "Address with wrong provider data type",
			record: libdns.Address{IP: netip.MustParseAddr("1.2.3.4"), ProviderData: "not-providerData"},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRecordID(tt.record)
			if got != tt.want {
				t.Errorf("recordID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetRecordID(t *testing.T) {
	tests := []struct {
		name   string
		record libdns.Record
		id     string
	}{
		{"Address", libdns.Address{IP: netip.MustParseAddr("1.2.3.4")}, "100"},
		{"CNAME", libdns.CNAME{Target: "example.com."}, "200"},
		{"MX", libdns.MX{Target: "mail.example.com."}, "300"},
		{"NS", libdns.NS{Target: "ns1.example.com."}, "400"},
		{"TXT", libdns.TXT{Text: "hello"}, "500"},
		{"SRV", libdns.SRV{Target: "sip.example.com."}, "600"},
		{"CAA", libdns.CAA{Tag: "issue", Value: "ca.example.com"}, "700"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := setRecordID(tt.record, tt.id)
			got := getRecordID(result)
			if got != tt.id {
				t.Errorf("after setRecordID(%q), recordID() = %q", tt.id, got)
			}
		})
	}

	t.Run("unsupported type returns original", func(t *testing.T) {
		rr := libdns.RR{Name: "test", Type: "UNKNOWN", Data: "data"}
		result := setRecordID(rr, "999")
		if getRecordID(result) != "" {
			t.Error("expected empty ID for unsupported record type")
		}
	})
}

func TestRecordToLibDNSRecord(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		record   models.DNSRecord
		wantType string
		wantName string
		wantID   string
		wantPrio uint16
		wantData string
		wantErr  bool
	}{
		{
			name:   "A record",
			domain: "example.com",
			record: models.DNSRecord{
				ID:      "10",
				Name:    "sub.example.com",
				Type:    "A",
				Content: "1.2.3.4",
				TTL:     "3600",
			},
			wantType: "A",
			wantName: "sub",
			wantID:   "10",
		},
		{
			name:   "MX record",
			domain: "example.com",
			record: models.DNSRecord{
				ID:       "20",
				Name:     "example.com",
				Type:     "MX",
				Content:  "mail.example.com",
				TTL:      "300",
				Priority: "10",
			},
			wantType: "MX",
			wantName: "@",
			wantID:   "20",
			wantPrio: 10,
		},
		{
			name:   "MX record with invalid priority",
			domain: "example.com",
			record: models.DNSRecord{
				ID:       "21",
				Name:     "example.com",
				Type:     "MX",
				Content:  "not a hostname and more",
				TTL:      "300",
				Priority: "abc",
			},
			wantErr: true,
		},
		{
			name:   "SRV record",
			domain: "example.com",
			record: models.DNSRecord{
				ID:       "25",
				Name:     "_sip._tcp.example.com",
				Type:     "SRV",
				Content:  "60 5060 sip.example.com",
				TTL:      "3600",
				Priority: "10",
			},
			wantType: "SRV",
			wantName: "_sip._tcp",
			wantID:   "25",
			wantPrio: 10,
		},
		{
			name:   "CNAME record regains trailing dot",
			domain: "example.com",
			record: models.DNSRecord{
				ID:      "26",
				Name:    "www.example.com",
				Type:    "CNAME",
				Content: "target.example.com",
				TTL:     "3600",
			},
			wantType: "CNAME",
			wantName: "www",
			wantID:   "26",
			wantData: "target.example.com.",
		},
		{
			name:   "NS record regains trailing dot",
			domain: "example.com",
			record: models.DNSRecord{
				ID:      "27",
				Name:    "sub.example.com",
				Type:    "NS",
				Content: "ns1.example.com",
				TTL:     "3600",
			},
			wantType: "NS",
			wantName: "sub",
			wantID:   "27",
			wantData: "ns1.example.com.",
		},
		{
			name:   "TXT record",
			domain: "example.com",
			record: models.DNSRecord{
				ID:      "30",
				Name:    "example.com",
				Type:    "TXT",
				Content: "v=spf1 include:example.com ~all",
				TTL:     "3600",
			},
			wantType: "TXT",
			wantName: "@",
			wantID:   "30",
		},
		{
			name:   "invalid TTL defaults to 0",
			domain: "example.com",
			record: models.DNSRecord{
				ID:      "40",
				Name:    "example.com",
				Type:    "A",
				Content: "5.6.7.8",
				TTL:     "invalid",
			},
			wantType: "A",
			wantName: "@",
			wantID:   "40",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fromSiteHostRecord(tt.domain, tt.record)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			rr := result.RR()
			if rr.Type != tt.wantType {
				t.Errorf("type = %q, want %q", rr.Type, tt.wantType)
			}
			if rr.Name != tt.wantName {
				t.Errorf("name = %q, want %q", rr.Name, tt.wantName)
			}
			if got := getRecordID(result); got != tt.wantID {
				t.Errorf("recordID = %q, want %q", got, tt.wantID)
			}
			if tt.wantData != "" && rr.Data != tt.wantData {
				t.Errorf("data = %q, want %q", rr.Data, tt.wantData)
			}
			if tt.wantPrio != 0 {
				if mx, ok := result.(libdns.MX); ok {
					if mx.Preference != tt.wantPrio {
						t.Errorf("priority = %d, want %d", mx.Preference, tt.wantPrio)
					}
				} else if srv, ok := result.(libdns.SRV); ok {
					if srv.Priority != tt.wantPrio {
						t.Errorf("priority = %d, want %d", srv.Priority, tt.wantPrio)
					}
				} else {
					t.Error("expected MX or SRV record type")
				}
			}
		})
	}

	t.Run("valid TTL is parsed", func(t *testing.T) {
		result, err := fromSiteHostRecord("example.com", models.DNSRecord{
			ID: "50", Name: "example.com", Type: "A", Content: "1.2.3.4", TTL: "7200",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rr := result.RR(); rr.TTL != 7200*time.Second {
			t.Errorf("TTL = %v, want %v", rr.TTL, 7200*time.Second)
		}
	})
}

func TestToSiteHostRecord(t *testing.T) {
	tests := []struct {
		name    string
		record  libdns.Record
		want    string
		wantErr bool
	}{
		{
			name:   "A record",
			record: libdns.Address{Name: "www", IP: netip.MustParseAddr("1.2.3.4")},
			want:   "1.2.3.4",
		},
		{
			name:   "CNAME record",
			record: libdns.CNAME{Name: "www", Target: "target.example.com."},
			want:   "target.example.com.",
		},
		{
			name:   "NS record",
			record: libdns.NS{Name: "sub", Target: "ns1.example.com."},
			want:   "ns1.example.com.",
		},
		{
			name:   "TXT record",
			record: libdns.TXT{Name: "@", Text: "v=spf1 -all"},
			want:   "v=spf1 -all",
		},
		{
			name:   "CAA record",
			record: libdns.CAA{Name: "@", Tag: "issue", Value: "letsencrypt.org"},
			want:   `0 issue "letsencrypt.org"`,
		},
		{
			name:   "MX record drops priority and trailing dot",
			record: libdns.MX{Name: "@", Preference: 10, Target: "mail.example.com."},
			want:   "mail.example.com",
		},
		{
			name:   "MX record without trailing dot",
			record: libdns.MX{Name: "@", Preference: 10, Target: "mail.example.com"},
			want:   "mail.example.com",
		},
		{
			name: "SRV record drops priority and trailing dot",
			record: libdns.SRV{
				Service: "sip", Transport: "tcp", Name: "@",
				Priority: 10, Weight: 5, Port: 5060, Target: "sip.example.com.",
			},
			want: "5 5060 sip.example.com",
		},
		{
			name:    "MX record with no fields set",
			record:  libdns.MX{Name: "@"},
			wantErr: true,
		},
		{
			name:    "MX record with no target",
			record:  libdns.MX{Name: "@", Preference: 10},
			wantErr: true,
		},
		{
			name:    "MX record with null target",
			record:  libdns.MX{Name: "@", Target: "."},
			wantErr: true,
		},
		{
			name: "SRV record with no target",
			record: libdns.SRV{
				Service: "sip", Transport: "tcp", Name: "@",
				Priority: 10, Weight: 5, Port: 5060,
			},
			wantErr: true,
		},
		{
			name: "SRV record with null target",
			record: libdns.SRV{
				Service: "sip", Transport: "tcp", Name: "@", Target: ".",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toSiteHostRecord(tt.record)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got content %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("content = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRecords(t *testing.T) {
	t.Run("generic records become typed", func(t *testing.T) {
		parsed, err := parseRecords([]libdns.Record{
			libdns.RR{Name: "@", Type: "MX", Data: "10 mail.example.com."},
			libdns.RR{Name: "_sip._tcp", Type: "SRV", Data: "10 5 5060 sip.example.com."},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mx, ok := parsed[0].(libdns.MX)
		if !ok {
			t.Fatalf("parsed[0] = %T, want libdns.MX", parsed[0])
		}
		if mx.Preference != 10 || mx.Target != "mail.example.com." {
			t.Errorf("MX = %+v, want preference 10 and target mail.example.com.", mx)
		}
		if got := recordPriority(parsed[1]); got != 10 {
			t.Errorf("SRV priority = %d, want 10", got)
		}

		// A typed record can carry a SiteHost record ID, which SetRecords
		// relies on when updating an existing record in place.
		if got := getRecordID(setRecordID(parsed[0], "123")); got != "123" {
			t.Errorf("recordID = %q, want %q", got, "123")
		}
	})

	t.Run("typed records are unchanged", func(t *testing.T) {
		input := libdns.TXT{Name: "@", Text: "hello", ProviderData: providerData{ID: "7"}}
		parsed, err := parseRecords([]libdns.Record{input})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if parsed[0] != libdns.Record(input) {
			t.Errorf("parsed[0] = %+v, want %+v", parsed[0], input)
		}
	})

	t.Run("malformed generic record", func(t *testing.T) {
		if _, err := parseRecords([]libdns.Record{
			libdns.RR{Name: "@", Type: "MX", Data: "not-a-priority mail.example.com."},
		}); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
