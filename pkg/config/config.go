package config

import (
	"fmt"
	"os"
	"time"

	"github.com/axatol/gonfig"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var Values = struct {
	LogLevel  string `env:"LOG_LEVEL"  flag:"log-level"  default:"info"`
	LogFormat string `env:"LOG_FORMAT" flag:"log-format" default:"json"`

	CloudflareAPIKey    string `env:"CLOUDFLARE_API_KEY"    flag:"cloudflare-api-key"`
	CloudflareAPIEmail  string `env:"CLOUDFLARE_API_EMAIL"  flag:"cloudflare-api-email"`
	CloudflareAPIToken  string `env:"CLOUDFLARE_API_TOKEN"  flag:"cloudflare-api-token"`
	CloudflareAccountID string `env:"CLOUDFLARE_ACCOUNT_ID" flag:"cloudflare-account-id" required:"true"`
	CloudflareTunnelID  string `env:"CLOUDFLARE_TUNNEL_ID"  flag:"cloudflare-tunnel-id"   required:"true"`

	Port         int64         `env:"PORT"          flag:"port"          default:"8888"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT"  flag:"read-timeout"  default:"5s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" flag:"write-timeout" default:"10s"`
	DryRun       bool          `env:"DRY_RUN"       flag:"dry-run"       default:"false"`
	DomainFilter []string      `env:"DOMAIN_FILTER" flag:"domain-filter" delimiter:","`
}{}

func Configure() error {
	// ignore error if .env file does not exist
	_ = godotenv.Load()

	if err := gonfig.Load(&Values); err != nil {
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
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	default:
		return fmt.Errorf("invalid log format: %s", Values.LogFormat)
	}

	log.Logger = log.Logger.With().Caller().Stack().Logger()
	zerolog.DefaultContextLogger = &log.Logger

	return nil
}
