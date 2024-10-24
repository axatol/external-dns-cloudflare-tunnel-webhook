package cf

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type cloudflareLogger struct {
	logger zerolog.Logger
}

func (l cloudflareLogger) Printf(format string, v ...any) {
	log.Debug().Str("context", "cloudflareAPI").Msgf(format, v...)
}

type Cloudflare interface {
	GetTunnelConfiguration(ctx context.Context, accountID, tunnelID string) (*cloudflare.TunnelConfigurationResult, error)
	UpdateTunnelIngress(ctx context.Context, accountID, tunnelID string, ingress []cloudflare.UnvalidatedIngressRule) error

	ListZones(ctx context.Context) ([]cloudflare.Zone, error)
	ListAllZoneRecords(ctx context.Context) ([]cloudflare.DNSRecord, error)
	ListZoneRecords(ctx context.Context, zoneID string) ([]cloudflare.DNSRecord, error)

	CreateDNSRecord(ctx context.Context, record cloudflare.DNSRecord) error
	DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error
	UpdateDNSRecord(ctx context.Context, record cloudflare.DNSRecord) error
}

func NewCloudflareClient(email, key, token string) (Cloudflare, error) {
	client := clientImpl{}

	if key == "" && token == "" {
		return nil, fmt.Errorf("either CLOUDFLARE_API_KEY or CLOUDFLARE_API_TOKEN must be set")
	}

	if key != "" && email == "" {
		return nil, fmt.Errorf("CLOUDFLARE_API_EMAIL must be set when using CLOUDFLARE_API_KEY")
	}

	options := []cloudflare.Option{
		cloudflare.UsingLogger(cloudflareLogger{log.Logger}),
	}

	var err error
	if token != "" {
		client.api, err = cloudflare.NewWithAPIToken(token, options...)
	} else {
		client.api, err = cloudflare.New(key, email, options...)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create cloudflare client: %w", err)
	}

	return &client, nil
}

type clientImpl struct{ api *cloudflare.API }

func (p clientImpl) GetTunnelConfiguration(ctx context.Context, accountID, tunnelID string) (*cloudflare.TunnelConfigurationResult, error) {
	rc := cloudflare.ResourceIdentifier(accountID)
	tunnel, err := p.api.GetTunnelConfiguration(ctx, rc, tunnelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tunnel configuration: %w", err)
	}

	log.Debug().Any("tunnel_configuration", tunnel).Send()
	return &tunnel, nil
}

func (p clientImpl) UpdateTunnelIngress(ctx context.Context, accountID, tunnelID string, ingress []cloudflare.UnvalidatedIngressRule) error {
	rc := cloudflare.ResourceIdentifier(accountID)

	tunnel, err := p.api.GetTunnelConfiguration(ctx, rc, tunnelID)
	if err != nil {
		return fmt.Errorf("failed to get tunnel configuration: %w", err)
	}

	log.Trace().Any("tunnel_before", tunnel).Send()

	tunnel.Config.Ingress = ingress
	params := cloudflare.TunnelConfigurationParams{
		TunnelID: tunnelID,
		Config:   tunnel.Config,
	}

	log.Trace().Any("tunnel_after", tunnel).Send()

	tunnel, err = p.api.UpdateTunnelConfiguration(ctx, rc, params)
	if err != nil {
		return fmt.Errorf("failed to update tunnel configuration: %w", err)
	}

	log.Debug().Any("updated_tunnel_configuration", tunnel).Send()
	return nil
}

func (p clientImpl) ListZones(ctx context.Context) ([]cloudflare.Zone, error) {
	zones, err := p.api.ListZones(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	log.Debug().Any("zones", zones).Send()
	return zones, nil
}

func (p clientImpl) ListAllZoneRecords(ctx context.Context) ([]cloudflare.DNSRecord, error) {
	zones, err := p.ListZones(ctx)
	if err != nil {
		return nil, err
	}

	records := []cloudflare.DNSRecord{}
	for _, zone := range zones {
		zoneRecords, err := p.ListZoneRecords(ctx, zone.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get zone records: %w", err)
		}

		records = append(records, zoneRecords...)
	}

	return records, nil
}

func (p clientImpl) ListZoneRecords(ctx context.Context, zoneID string) ([]cloudflare.DNSRecord, error) {
	rc := cloudflare.ZoneIdentifier(zoneID)
	records, _, err := p.api.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list dns records for zone %s: %w", zoneID, err)
	}

	log.Debug().Any("records", records).Send()
	return records, nil
}

func (p clientImpl) CreateDNSRecord(ctx context.Context, record cloudflare.DNSRecord) error {
	rc := cloudflare.ZoneIdentifier(record.ZoneID)
	params := cloudflare.CreateDNSRecordParams{
		Name:    record.Name,
		Type:    record.Type,
		Proxied: record.Proxied,
		TTL:     record.TTL,
		Comment: record.Comment,
		Content: record.Content,
	}

	record, err := p.api.CreateDNSRecord(ctx, rc, params)
	if err != nil {
		return fmt.Errorf("failed to create dns record %s: %w", record.Name, err)
	}

	log.Debug().Any("created_record", record).Send()
	return nil
}

func (p clientImpl) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	rc := cloudflare.ZoneIdentifier(zoneID)
	if err := p.api.DeleteDNSRecord(ctx, rc, recordID); err != nil {
		return fmt.Errorf("failed to delete dns record %s: %w", recordID, err)
	}

	log.Debug().Str("deleted_record_id", recordID).Send()
	return nil
}

func (p clientImpl) UpdateDNSRecord(ctx context.Context, record cloudflare.DNSRecord) error {
	rc := cloudflare.ZoneIdentifier(record.ZoneID)
	params := cloudflare.UpdateDNSRecordParams{
		ID:      record.ID,
		Name:    record.Name,
		Type:    record.Type,
		Proxied: record.Proxied,
		TTL:     record.TTL,
		Comment: cloudflare.StringPtr(record.Comment),
		Content: record.Content,
	}

	record, err := p.api.UpdateDNSRecord(ctx, rc, params)
	if err != nil {
		return fmt.Errorf("failed to update dns record %s: %w", record.Name, err)
	}

	log.Debug().Any("updated_record", record).Send()
	return nil
}
