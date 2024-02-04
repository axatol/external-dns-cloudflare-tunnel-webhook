package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/axatol/external-dns-cloudflare-tunnel-webhook/pkg/cf"
	"github.com/axatol/external-dns-cloudflare-tunnel-webhook/pkg/util"
	"github.com/cloudflare/cloudflare-go"
	"sigs.k8s.io/external-dns/endpoint"
)

type ChangeType string

const (
	ChangeTypeNoop   ChangeType = "NOOP"
	ChangeTypeCreate ChangeType = "CREATE"
	ChangeTypeUpdate ChangeType = "UPDATE"
	ChangeTypeDelete ChangeType = "DELETE"
)

type Change struct {
	Action    ChangeType
	ZoneID    string
	RecordID  string
	Name      string
	TunnelURI string
	Service   string
}

type ZoneDetail struct {
	Zone    cloudflare.Zone
	Records map[string]cloudflare.DNSRecord
}

type ZoneMap map[string]ZoneDetail

// GetMatchingZone finds the longest matching zone for the hostname
func (z ZoneMap) GetMatchingZone(hostname string) *ZoneDetail {
	longestName := ""
	for name := range z {
		if len(name) > len(longestName) && strings.HasSuffix(hostname, name) {
			longestName = name
		}
	}

	if matching, ok := z[longestName]; ok {
		return &matching
	}

	return nil
}

// GetRecordByName finds a record by name in the zone map
func (z ZoneMap) GetRecordByName(hostname string) *cloudflare.DNSRecord {
	for _, zone := range z {
		if record, ok := zone.Records[hostname]; ok {
			return &record
		}
	}

	return nil
}

func GenerateZoneMap(ctx context.Context, cf cf.Cloudflare) (*ZoneMap, error) {
	zones, err := cf.ListZones(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %s", err)
	}

	zoneMap := ZoneMap{}
	for _, zone := range zones {
		records, err := cf.ListZoneRecords(ctx, zone.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get zone records: %s", err)
		}

		recordMap := map[string]cloudflare.DNSRecord{}
		for _, record := range records {
			recordMap[record.Name] = record
		}

		zoneMap[zone.Name] = ZoneDetail{zone, recordMap}
	}

	return &zoneMap, nil
}

func TunnelDNSChangeSet(tunnelID string, rules []cloudflare.UnvalidatedIngressRule, zoneMap ZoneMap) []Change {
	tunnelURI := fmt.Sprintf("%s.cfargotunnel.com", tunnelID)

	ruleMap := map[string]cloudflare.UnvalidatedIngressRule{}
	for _, rule := range rules {
		ruleMap[rule.Hostname] = rule
	}

	changes := map[string]Change{}
	for _, rule := range rules {
		change := Change{
			Action:    ChangeTypeNoop,
			Name:      rule.Hostname,
			TunnelURI: tunnelURI,
			Service:   rule.Service,
		}

		record := zoneMap.GetRecordByName(rule.Hostname)
		if record == nil {
			zone := zoneMap.GetMatchingZone(rule.Hostname)
			if zone == nil {
				continue
			}

			change.Action = ChangeTypeCreate
			change.ZoneID = zone.Zone.ID
			changes[rule.Hostname] = change
			continue
		}

		if record.Content == tunnelURI {
			changes[rule.Hostname] = change
			continue
		}

		change.Action = ChangeTypeUpdate
		change.ZoneID = record.ZoneID
		change.RecordID = record.ID
		changes[rule.Hostname] = change
	}

	for _, zone := range zoneMap {
		for name, record := range zone.Records {
			if _, ok := ruleMap[name]; !ok && record.Content == tunnelURI {
				changes[name] = Change{
					Action:    ChangeTypeDelete,
					ZoneID:    record.ZoneID,
					RecordID:  record.ID,
					Name:      record.Name,
					TunnelURI: record.Content,
				}
			}
		}
	}

	changeList := make([]Change, 0, len(changes))
	for _, change := range changes {
		if change.Action != ChangeTypeNoop {
			changeList = append(changeList, change)
		}
	}

	return changeList
}

func ApplyChanges(ctx context.Context, cf cf.Cloudflare, changes []Change) error {
	errs := util.ErrorList{}

	for _, change := range changes {
		record := cloudflare.DNSRecord{
			ID:      change.RecordID,
			ZoneID:  change.ZoneID,
			Name:    change.Name,
			Content: change.TunnelURI,
			Type:    endpoint.RecordTypeCNAME,
			TTL:     1,
			Proxied: cloudflare.BoolPtr(true),
			Comment: fmt.Sprintf("external-dns-cloudflare-tunnel-webhook/%s", change.Service),
		}

		switch change.Action {
		case ChangeTypeCreate:
			if err := cf.CreateDNSRecord(ctx, record); err != nil {
				errs = append(errs, err)
			}

		case ChangeTypeUpdate:
			if err := cf.UpdateDNSRecord(ctx, record); err != nil {
				errs = append(errs, err)
			}

		case ChangeTypeDelete:
			if err := cf.DeleteDNSRecord(ctx, change.ZoneID, change.RecordID); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return &errs
	}

	return nil
}

func BatchUpdateDNSRecords(ctx context.Context, cf cf.Cloudflare, changes []Change) error {
	errs := util.ErrorList{}

	for _, change := range changes {
		record := cloudflare.DNSRecord{
			ID:      change.RecordID,
			ZoneID:  change.ZoneID,
			Name:    change.Name,
			Content: change.TunnelURI,
			Type:    endpoint.RecordTypeCNAME,
			TTL:     1,
			Proxied: cloudflare.BoolPtr(true),
			Comment: fmt.Sprintf("external-dns-cloudflare-tunnel-webhook/%s", change.Service),
		}

		switch change.Action {
		case ChangeTypeCreate:
			if err := cf.CreateDNSRecord(ctx, record); err != nil {
				errs = append(errs, err)
			}

		case ChangeTypeUpdate:
			if err := cf.UpdateDNSRecord(ctx, record); err != nil {
				errs = append(errs, err)
			}

		case ChangeTypeDelete:
			if err := cf.DeleteDNSRecord(ctx, change.ZoneID, change.RecordID); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return &errs
	}

	return nil
}
