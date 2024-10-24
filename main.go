package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/axatol/external-dns-cloudflare-tunnel-webhook/pkg/cf"
	"github.com/axatol/external-dns-cloudflare-tunnel-webhook/pkg/config"
	"github.com/axatol/external-dns-cloudflare-tunnel-webhook/pkg/provider"
	"github.com/axatol/external-dns-cloudflare-tunnel-webhook/pkg/server"
	"github.com/rs/zerolog/log"
)

var (
	buildCommit = "unknown"
	buildTime   = "unknown"

	build = map[string]any{
		"go_os":        runtime.GOOS,
		"go_arch":      runtime.GOARCH,
		"go_version":   runtime.Version(),
		"build_commit": buildCommit,
		"build_time":   buildTime,
	}
)

func main() {
	if err := config.Configure(); err != nil {
		log.Fatal().Err(err).Fields(build).Send()
	}

	log.Info().
		Fields(build).
		Str("log_level", config.Values.LogLevel).
		Str("log_format", config.Values.LogFormat).
		Str("cloudflare_api_key", strings.Repeat("*", len(config.Values.CloudflareAPIKey))).
		Str("cloudflare_api_email", strings.Repeat("*", len(config.Values.CloudflareAPIEmail))).
		Str("cloudflare_api_token", strings.Repeat("*", len(config.Values.CloudflareAPIToken))).
		Str("cloudflare_account_id", config.Values.CloudflareAccountID).
		Str("cloudflare_tunnel_id", config.Values.CloudflareTunnelID).
		Int64("port", config.Values.Port).
		Dur("read_timeout", config.Values.ReadTimeout).
		Dur("write_timeout", config.Values.WriteTimeout).
		Bool("dry_run", config.Values.DryRun).
		Strs("domain_filter", config.Values.DomainFilter).
		Send()

	client, err := cf.NewCloudflareClient(config.Values.CloudflareAPIEmail, config.Values.CloudflareAPIKey, config.Values.CloudflareAPIToken)
	if err != nil {
		log.Fatal().Err(fmt.Errorf("failed to create cloudflare client: %w", err)).Send()
	}

	provider := provider.CloudflareTunnelProvider{
		Cloudflare:          client,
		CloudflareAccountID: config.Values.CloudflareAccountID,
		CloudflareTunnelID:  config.Values.CloudflareTunnelID,
		DryRun:              config.Values.DryRun,
		DomainFilter:        config.Values.DomainFilter,
	}

	if err != nil {
		log.Fatal().Err(fmt.Errorf("failed to create provider: %w", err)).Send()
	}

	server := server.NewServer(config.Values.Port, provider, config.Values.ReadTimeout, config.Values.WriteTimeout)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(fmt.Errorf("failed to start server: %w", err)).Send()
			cancel()
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down server")

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	go func() {
		defer cancel()
		if err := server.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
			log.Error().Err(fmt.Errorf("failed to shutdown server: %w", err)).Send()
		}
	}()

	<-ctx.Done()
	if err := ctx.Err(); err != nil && err != context.Canceled {
		log.Error().Err(fmt.Errorf("failed to shutdown server: %w", err)).Send()
	}
}
