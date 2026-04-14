// Package sitehost implements a provider for
// [libdns](https://github.com/libdns/libdns) to manage DNS records which are
// hosted by [SiteHost](https://sitehost.nz/).
package sitehost

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/libdns/libdns"
	"github.com/sitehostnz/gosh/pkg/api"
	"github.com/sitehostnz/gosh/pkg/api/dns"
)

// Provider implements the libdns interfaces for managing records that are
// hosted by SiteHost.
type Provider struct {
	// ClientID is the ID of the SiteHost client this provider should act as
	// when performing operations on zones.
	ClientID string `json:"client_id,omitempty"`
	// APIKey is the key this provider should use to authenticate with the
	// SiteHost API.
	APIKey string `json:"api_key,omitempty"`
	// APIHost optionally overrides the hostname this provider should use to
	// connect to the SiteHost API. If empty, the production instance at
	// `api.sitehost.nz` is used.
	APIHost string `json:"api_host,omitempty"`

	dnsClient  *dns.Client
	clientOnce sync.Once
}

// getClient returns an instance of the SiteHost DNS client.
func (p *Provider) getClient() *dns.Client {
	// Ensure that initialisation only happens once in the event that there are
	// several concurrent operations happening.
	p.clientOnce.Do(func() {
		client := api.NewClient(p.APIKey, p.ClientID)
		if p.APIHost != "" {
			// NOTE: The API version in `Path` should be updated whenever a new
			// API version is released.
			client.BaseURL = &url.URL{
				Scheme: "https",
				Host:   p.APIHost,
				Path:   "/1.5/",
			}
		}

		p.dnsClient = dns.New(client)
	})

	return p.dnsClient
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	zone = strings.Trim(zone, ".")

	return p.getRecords(ctx, zone)
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	zone = strings.Trim(zone, ".")

	return p.createRecords(ctx, zone, records)
}

// DeleteRecords deletes the records from the zone. If a record does not have an ID,
// it will be looked up. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	var deleted []libdns.Record
	var deleteQueue []libdns.Record

	zone = strings.Trim(zone, ".")

	// Fetch zone records once if any input record needs a lookup.
	var zoneRecords []libdns.Record
	for _, rec := range recs {
		if getRecordID(rec) == "" {
			var err error
			zoneRecords, err = p.getRecords(ctx, zone)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	for _, rec := range recs {
		if getRecordID(rec) != "" {
			deleteQueue = append(deleteQueue, rec)
			continue
		}
		deleteQueue = append(deleteQueue, deleteMatchRecords(zoneRecords, rec)...)
	}

	for _, rec := range deleteQueue {
		if err := p.deleteRecord(ctx, zone, rec); err != nil {
			return deleted, err
		}
		deleted = append(deleted, rec)
	}

	return deleted, nil
}

// SetRecords sets the records in the zone, either by updating existing records
// or creating new ones.
//
// For each (name, type) pair in the input, all existing records with that pair
// are replaced by the input records. It returns the records that were set.
//
// Operations are not atomic: if an error occurs partway through, any records
// that were successfully set before the failure are returned alongside the
// error so callers can reason about the partial state of the zone.
func (p *Provider) SetRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	var results []libdns.Record

	zone = strings.Trim(zone, ".")

	zoneRecords, err := p.getRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	// Group input and existing records by (name, type).
	type nameType struct{ name, typ string }

	inputGroups := make(map[nameType][]libdns.Record)
	var groupOrder []nameType
	for _, rec := range recs {
		rr := rec.RR()
		nt := nameType{rr.Name, rr.Type}
		if _, exists := inputGroups[nt]; !exists {
			groupOrder = append(groupOrder, nt)
		}
		inputGroups[nt] = append(inputGroups[nt], rec)
	}

	existingGroups := make(map[nameType][]libdns.Record)
	for _, rec := range zoneRecords {
		rr := rec.RR()
		nt := nameType{rr.Name, rr.Type}
		if _, affected := inputGroups[nt]; affected {
			existingGroups[nt] = append(existingGroups[nt], rec)
		}
	}

	for _, nt := range groupOrder {
		inputs := inputGroups[nt]
		existing := existingGroups[nt]

		// When there is exactly one record on each side, update in place
		// rather than deleting and recreating. Use the input's ID if it
		// already has one, otherwise adopt the existing record's ID. If
		// the IDs are both present but disagree, fall through to the
		// general case.
		if len(inputs) == 1 && len(existing) == 1 {
			input := inputs[0]
			inputID := getRecordID(input)
			existingID := getRecordID(existing[0])

			if inputID == "" || inputID == existingID {
				result, err := p.updateRecord(ctx, zone, setRecordID(input, existingID))
				if err != nil {
					return results, err
				}
				results = append(results, result)
				continue
			}
		}

		// General case: delete all existing records for this (name, type)
		// and create the input records fresh.
		for _, rec := range existing {
			if err := p.deleteRecord(ctx, zone, rec); err != nil {
				return results, err
			}
		}
		for _, rec := range inputs {
			result, err := p.createRecord(ctx, zone, rec)
			if err != nil {
				return results, err
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// createRecords creates multiple DNS records and returns them with their IDs.
func (p *Provider) createRecords(ctx context.Context, domain string, recs []libdns.Record) ([]libdns.Record, error) {
	var created []libdns.Record
	for _, rec := range recs {
		result, err := p.createRecord(ctx, domain, rec)
		if err != nil {
			return created, err
		}
		created = append(created, result)
	}
	return created, nil
}

// createRecord creates a DNS record and returns it with its ID.
func (p *Provider) createRecord(ctx context.Context, domain string, rec libdns.Record) (libdns.Record, error) {
	client := p.getClient()

	rr := rec.RR()
	priority := uint16(0)
	if mx, ok := rec.(libdns.MX); ok {
		priority = mx.Preference
	} else if srv, ok := rec.(libdns.SRV); ok {
		priority = srv.Priority
	}

	response, err := client.AddRecord(ctx, dns.AddRecordRequest{
		Domain:   domain,
		Type:     rr.Type,
		Name:     libdns.AbsoluteName(rr.Name, domain),
		Content:  toSiteHostRecord(rec),
		Priority: strconv.FormatUint(uint64(priority), 10),
	})
	if err != nil {
		return nil, err
	}

	if !response.Status {
		return nil, errors.New(response.Msg)
	}

	return setRecordID(rec, response.Return.ID), nil
}

// deleteRecord deletes a DNS record from the zone.
func (p *Provider) deleteRecord(ctx context.Context, domain string, rec libdns.Record) error {
	client := p.getClient()
	response, err := client.DeleteRecord(ctx, dns.DeleteRecordRequest{
		Domain:   domain,
		RecordID: getRecordID(rec),
	})
	if err != nil {
		return err
	}

	if !response.Status {
		return errors.New(response.Msg)
	}

	return nil
}

// getRecords retrieves all DNS records for the given domain.
func (p *Provider) getRecords(ctx context.Context, domain string) ([]libdns.Record, error) {
	client := p.getClient()
	response, err := client.ListRecords(ctx, dns.ListRecordsRequest{
		Domain: domain,
	})
	if err != nil {
		return nil, err
	}

	records := make([]libdns.Record, 0, len(response.Return))
	for _, record := range response.Return {
		r, err := fromSiteHostRecord(domain, record)
		if err != nil {
			// Skip records libdns has no typed form for (e.g. SOA). They
			// cannot be round-tripped via this provider, but we must not
			// fail the whole listing because of them.
			if errors.Is(err, errUnsupportedRecordType) {
				continue
			}

			return nil, err
		}

		records = append(records, r)
	}

	return records, nil
}

// updateRecord updates a DNS record and returns it.
func (p *Provider) updateRecord(ctx context.Context, domain string, rec libdns.Record) (libdns.Record, error) {
	client := p.getClient()

	rr := rec.RR()
	priority := uint16(0)
	if mx, ok := rec.(libdns.MX); ok {
		priority = mx.Preference
	} else if srv, ok := rec.(libdns.SRV); ok {
		priority = srv.Priority
	}

	response, err := client.UpdateRecord(ctx, dns.UpdateRecordRequest{
		Domain:   domain,
		RecordID: getRecordID(rec),
		Name:     libdns.AbsoluteName(rr.Name, domain),
		Type:     rr.Type,
		Content:  toSiteHostRecord(rec),
		Priority: strconv.FormatUint(uint64(priority), 10),
	})
	if err != nil {
		return nil, err
	}

	if !response.Status {
		return nil, errors.New(response.Msg)
	}

	return rec, nil
}

// ListZones returns all DNS zones available on the account.
func (p *Provider) ListZones(ctx context.Context) ([]libdns.Zone, error) {
	client := p.getClient()

	var zones []libdns.Zone
	page := 1

	for {
		response, err := client.ListZones(ctx, &dns.ListZoneOptions{
			PageNumber: page,
		})
		if err != nil {
			return nil, err
		}

		for _, z := range response.Return.Data {
			zones = append(zones, libdns.Zone{Name: z.Name + "."})
		}

		if page >= response.Return.TotalPages {
			break
		}
		page++
	}

	return zones, nil
}

// Verify that the `Provider` struct actually implements the `libdns`
// interfaces.
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
	_ libdns.ZoneLister     = (*Provider)(nil)
)
