package sitehost

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/libdns/libdns"
	"github.com/sitehostnz/gosh/pkg/models"
)

// errUnsupportedRecordType is returned by fromSiteHostRecord when libdns has
// no typed record for the SiteHost record type (e.g. SOA). Such records
// cannot carry a SiteHost record ID, so they are skipped when listing a zone
// rather than returned as records we cannot round-trip safely.
var errUnsupportedRecordType = errors.New("unsupported record type")

// providerData holds SiteHost-specific data attached to libdns records.
type providerData struct {
	// The ID that identifies this record via the SiteHost API.
	ID string
}

// getRecordID extracts the SiteHost record ID from a record's provider data if
// present. Returns an empty string otherwise. The generic libdns.RR fallback
// type has no ProviderData field and therefore cannot carry an ID.
func getRecordID(rec libdns.Record) string {
	var data any

	switch r := rec.(type) {
	case libdns.Address:
		data = r.ProviderData
	case libdns.CAA:
		data = r.ProviderData
	case libdns.CNAME:
		data = r.ProviderData
	case libdns.MX:
		data = r.ProviderData
	case libdns.NS:
		data = r.ProviderData
	case libdns.SRV:
		data = r.ProviderData
	case libdns.ServiceBinding:
		data = r.ProviderData
	case libdns.TXT:
		data = r.ProviderData
	default:
		return ""
	}

	if pd, ok := data.(providerData); ok {
		return pd.ID
	}

	return ""
}

// setRecordID returns a copy of a record with the given SiteHost record ID in
// its provider data.
func setRecordID(rec libdns.Record, id string) libdns.Record {
	data := providerData{ID: id}

	switch r := rec.(type) {
	case libdns.Address:
		r.ProviderData = data
		return r
	case libdns.CAA:
		r.ProviderData = data
		return r
	case libdns.CNAME:
		r.ProviderData = data
		return r
	case libdns.MX:
		r.ProviderData = data
		return r
	case libdns.NS:
		r.ProviderData = data
		return r
	case libdns.SRV:
		r.ProviderData = data
		return r
	case libdns.ServiceBinding:
		r.ProviderData = data
		return r
	case libdns.TXT:
		r.ProviderData = data
		return r
	}

	return rec
}

// toSiteHostRecord returns the record content formatted for the SiteHost API.
func toSiteHostRecord(rec libdns.Record) string {
	rr := rec.RR()

	switch rec.(type) {
	case libdns.MX, libdns.SRV:
		// For MX and SRV records, the priority is sent via a separate API
		// field, so it must be stripped from the record data.
		if _, rest, ok := strings.Cut(rr.Data, " "); ok {
			return strings.TrimSuffix(rest, ".")
		}
	}

	return rr.Data
}

// Given the domain the record should be taken as relative to,
// fromSiteHostRecord converts a record as formatted by the SiteHost API to a
// libdns record.
func fromSiteHostRecord(domain string, rec models.DNSRecord) (libdns.Record, error) {
	name := libdns.RelativeName(rec.Name, domain)

	ttl, err := strconv.Atoi(rec.TTL)
	if err != nil {
		ttl = 0
	}

	// The API strips trailing dots from hostnames. Restore them for record
	// types that contain domain names so that libdns gets properly qualified
	// FQDNs.
	content := rec.Content
	switch rec.Type {
	case "CNAME", "NS", "MX", "SRV":
		if !strings.HasSuffix(content, ".") {
			content += "."
		}
	}

	// The SiteHost API stores priority in a dedicated field. For MX and
	// SRV records, parse it and prepend its numeric form to the content
	// so libdns gets standard zone-file data.
	var priority uint64
	switch rec.Type {
	case "MX", "SRV":
		priority, err = strconv.ParseUint(rec.Priority, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid priority %q from SiteHost API for %s record %q: %w", rec.Priority, rec.Type, rec.Name, err)
		}
		content = strconv.FormatUint(priority, 10) + " " + content
	}

	rr := libdns.RR{
		Name: name,
		TTL:  time.Duration(ttl) * time.Second,
		Type: rec.Type,
		Data: content,
	}
	parsed, err := rr.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing %s record %q: %w", rec.Type, rec.Name, err)
	}

	// If Parse returned the generic libdns.RR fallback, libdns doesn't know
	// this record type. We can't attach a SiteHost record ID to it, which
	// would make later Set/Delete operations unreliable - signal to the
	// caller that this record should be skipped rather than returned.
	if _, isGeneric := parsed.(libdns.RR); isGeneric {
		return nil, fmt.Errorf("%w: %s record %q", errUnsupportedRecordType, rec.Type, rec.Name)
	}

	parsed = setRecordID(parsed, rec.ID)

	// MX and SRV records need the priority set from the dedicated API field.
	if mx, ok := parsed.(libdns.MX); ok {
		mx.Preference = uint16(priority)
		return mx, nil
	}
	if srv, ok := parsed.(libdns.SRV); ok {
		srv.Priority = uint16(priority)
		return srv, nil
	}

	return parsed, nil
}

// deleteMatchRecords returns records from a given list that match a target for
// deletion. Per the libdns spec, name is always required, while type, TTL, and
// content act as wildcards when zero-valued.
func deleteMatchRecords(recs []libdns.Record, target libdns.Record) []libdns.Record {
	rr := target.RR()

	var matches []libdns.Record
	for _, rec := range recs {
		rrr := rec.RR()

		if rrr.Name != rr.Name {
			continue
		}
		if rr.Type != "" && rrr.Type != rr.Type {
			continue
		}
		if rr.TTL != 0 && rrr.TTL != rr.TTL {
			continue
		}
		if rr.Data != "" && rrr.Data != rr.Data {
			continue
		}

		matches = append(matches, rec)
	}

	return matches
}
