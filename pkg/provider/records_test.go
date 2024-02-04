package provider_test

import (
	"testing"

	"github.com/axatol/external-dns-cloudflare-tunnel-webhook/pkg/provider"
	"github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/assert"
)

func TestTunnelDNSChangeSet(t *testing.T) {
	tunnelID := "tunnel123"

	rules := []cloudflare.UnvalidatedIngressRule{
		{Hostname: "noop.example.com", Service: "noop"},
		{Hostname: "create.example.com", Service: "create"},
		{Hostname: "update.example.com", Service: "update"},
		// {Hostname: "delete.example.com", Service: "delete"},
	}

	zoneMap := provider.ZoneMap{
		"example.com": provider.ZoneDetail{
			Zone: cloudflare.Zone{
				ID: "zone123",
			},
			Records: map[string]cloudflare.DNSRecord{
				"noop.example.com": {
					ID:      "record0",
					ZoneID:  "zone123",
					Name:    "noop.example.com",
					Content: "tunnel123.cfargotunnel.com",
				},
				// "create.example.com": {
				// 	ID:      "record1",
				// 	ZoneID:  "zone123",
				// 	Name:    "create.example.com",
				// 	Content: "tunnel123.cfargotunnel.com",
				// },
				"update.example.com": {
					ID:      "record2",
					ZoneID:  "zone123",
					Name:    "update.example.com",
					Content: "blah",
				},
				"delete.example.com": {
					ID:      "record3",
					ZoneID:  "zone123",
					Name:    "delete.example.com",
					Content: "tunnel123.cfargotunnel.com",
				},
			},
		},
	}

	expected := []provider.Change{
		{
			Action:    provider.ChangeTypeCreate,
			ZoneID:    "zone123",
			RecordID:  "",
			Name:      "create.example.com",
			TunnelURI: "tunnel123.cfargotunnel.com",
			Service:   "create",
		},
		{
			Action:    provider.ChangeTypeUpdate,
			ZoneID:    "zone123",
			RecordID:  "record2",
			Name:      "update.example.com",
			TunnelURI: "tunnel123.cfargotunnel.com",
			Service:   "update",
		},
		{
			Action:    provider.ChangeTypeDelete,
			ZoneID:    "zone123",
			RecordID:  "record3",
			Name:      "delete.example.com",
			TunnelURI: "tunnel123.cfargotunnel.com",
			Service:   "",
		},
	}

	actual := provider.TunnelDNSChangeSet(tunnelID, rules, zoneMap)
	assert.ElementsMatch(t, expected, actual)
}
