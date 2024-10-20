package provider

import (
	"context"
	"fmt"

	"github.com/axatol/external-dns-cloudflare-tunnel-webhook/pkg/cf"
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

var _ provider.Provider = (*CloudflareTunnelProvider)(nil)

type CloudflareTunnelProvider struct {
	Cloudflare          cf.Cloudflare
	CloudflareAccountID string
	CloudflareTunnelID  string
	CloudflareSyncDNS   bool
	DryRun              bool
	DomainFilter        []string
}

// Records returns the list of live DNS records
//
// required to satisfy the external-dns provider interface
func (p CloudflareTunnelProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	tunnel, err := p.Cloudflare.GetTunnelConfiguration(ctx, p.CloudflareAccountID, p.CloudflareTunnelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tunnel configuration: %w", err)
	}

	endpoints := []*endpoint.Endpoint{}
	for _, ingress := range tunnel.Config.Ingress {
		if ingress.Hostname == "" {
			continue
		}

		endpoints = append(endpoints, &endpoint.Endpoint{
			DNSName:    ingress.Hostname,
			RecordType: endpoint.RecordTypeCNAME,
			Targets:    []string{ingress.Service},
			RecordTTL:  endpoint.TTL(1),
		})
	}

	return endpoints, nil
}

// AdjustEndpoints adjusts a given set of endpoints
//
// required to satisfy the external-dns provider interface
func (CloudflareTunnelProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	adjusted := []*endpoint.Endpoint{}
	for _, e := range endpoints {
		if e.RecordType != endpoint.RecordTypeCNAME {
			continue
		}

		adjusted = append(adjusted, e)
	}

	return adjusted, nil
}

func (p CloudflareTunnelProvider) GetDomainFilter() endpoint.DomainFilter {
	return endpoint.NewDomainFilter(p.DomainFilter)
}

// ApplyChanges applies a given set of changes
//
// required to satisfy the external-dns provider interface
func (p CloudflareTunnelProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	tunnel, err := p.Cloudflare.GetTunnelConfiguration(ctx, p.CloudflareAccountID, p.CloudflareTunnelID)
	if err != nil {
		return fmt.Errorf("failed to get tunnel configuration: %w", err)
	}

	rules := Rules(tunnel.Config.Ingress)
	if err := rules.ApplyChanges(changes); err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	zoneMap, err := GenerateZoneMap(ctx, p.Cloudflare)
	if err != nil {
		return fmt.Errorf("failed to generate zone map: %w", err)
	}

	changeset := TunnelDNSChangeSet(p.CloudflareTunnelID, rules, *zoneMap)

	if p.DryRun {
		log.Info().Any("rules", rules).Any("records", changeset).Msg("dry run, not applying changes")
		return nil
	}

	if err := p.Cloudflare.UpdateTunnelIngress(ctx, p.CloudflareAccountID, p.CloudflareTunnelID, rules); err != nil {
		return fmt.Errorf("failed to update tunnel ingress rules: %w", err)
	}

	if p.CloudflareSyncDNS {
		if err := BatchUpdateDNSRecords(ctx, p.Cloudflare, changeset); err != nil {
			return fmt.Errorf("failed to update zone records: %w", err)
		}
	}

	return nil
}
