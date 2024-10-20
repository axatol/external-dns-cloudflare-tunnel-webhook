package config

import (
	"fmt"
	"os"
	"time"

	"github.com/codingconcepts/env"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var Values = struct {
	LogLevel  string `env:"LOG_LEVEL" default:"info"`
	LogFormat string `env:"LOG_FORMAT" default:"json"`

	CloudflareAPIKey    string `env:"CLOUDFLARE_API_KEY"`
	CloudflareAPIEmail  string `env:"CLOUDFLARE_API_EMAIL"`
	CloudflareAPIToken  string `env:"CLOUDFLARE_API_TOKEN"`
	CloudflareAccountID string `env:"CLOUDFLARE_ACCOUNT_ID" required:"true"`
	CloudflareTunnelID  string `env:"CLOUDFLARE_TUNNEL_ID" required:"true"`
	CloudflareSyncDNS   bool   `env:"CLOUDFLARE_SYNC_DNS" default:"false"`

	Port         int64         `env:"PORT" default:"8888"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" default:"5s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" default:"10s"`
	DryRun       bool          `env:"DRY_RUN" default:"false"`
	DomainFilter []string      `env:"DOMAIN_FILTER" delimiter:","`
}{}

func Configure() error {
	// ignore error if .env file does not exist
	_ = godotenv.Load()

	if err := env.Set(&Values); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logLevel, err := zerolog.ParseLevel(Values.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to parse log level: %w", err)
	}

	zerolog.SetGlobalLevel(logLevel)
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	switch Values.LogFormat {
	case "json":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	case "text":
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout})
	default:
		return fmt.Errorf("invalid log format: %s", Values.LogFormat)
	}

	log.Logger = log.Logger.With().Caller().Stack().Logger()
	zerolog.DefaultContextLogger = &log.Logger

	return nil
}
