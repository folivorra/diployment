package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type CoreConfig struct {
	Env Env `env:"APP_ENV" env-default:"local"` // local, dev

	MasterKey []byte `env:"MASTER_KEY" env-required:"true"`

	HTTP     HTTPConfig
	Postgres PostgresConfig
	Auth     AuthConfig
	GitHub   GitHubConfig
	Webhook  WebhookServiceConfig
}

type WebhookConfig struct {
	Env Env `env:"APP_ENV" env-default:"local"` // local, dev

	MasterKey []byte `env:"MASTER_KEY" env-required:"true"`

	HTTP     HTTPConfig
	Postgres PostgresConfig
	NATS     NATSConfig
}

type Env string

const (
	EnvLocal Env = "local"
	EnvDev   Env = "dev"
)

func (e Env) IsValid() bool {
	return e == EnvLocal || e == EnvDev
}

type NATSConfig struct {
	URL string `env:"NATS_URL" env-required:"true"`
}

type HTTPConfig struct {
	Port string `env:"HTTP_PORT" env-default:"8080"`
	Host string `env:"HTTP_HOST" env-default:"localhost"`
}

func (h HTTPConfig) Address() string {
	return net.JoinHostPort(h.Host, h.Port)
}

type PostgresConfig struct {
	DSN string `env:"DATABASE_URL" env-required:"true"`
}

type AuthConfig struct {
	JWTSecret string        `env:"JWT_SECRET" env-required:"true"`
	JWTTTL    time.Duration `env:"JWT_TTL" env-default:"168h"`
}

type GitHubConfig struct {
	ClientID     string `env:"GITHUB_CLIENT_ID" env-required:"true"`
	ClientSecret string `env:"GITHUB_CLIENT_SECRET" env-required:"true"`
	RedirectURL  string `env:"GITHUB_REDIRECT_URL" env-required:"true"`
}

type WebhookServiceConfig struct {
	URL string `env:"WEBHOOK_URL" env-required:"true"`
}

func decodeBase64Key(encoded []byte) ([]byte, error) {
	key := make([]byte, len(encoded))
	_, err := base64.StdEncoding.Decode(key, encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding master key: %w", err)
	}
	return key, nil
}

func MustGetCore(envFile string) *CoreConfig {
	cfg := &CoreConfig{}
	if err := cleanenv.ReadConfig(envFile, cfg); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("cfg configure: %v", err)
		}
	}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		log.Fatalf("cfg configure: %v", err)
	}
	var err error
	if cfg.MasterKey, err = decodeBase64Key(cfg.MasterKey); err != nil {
		log.Fatalf("cfg configure: %v", err)
	}
	if !cfg.Env.IsValid() {
		log.Fatalf("cfg configure: APP_ENV invalid")
	}
	return cfg
}

type CoordinatorConfig struct {
	Env Env `env:"APP_ENV" env-default:"local"`

	Postgres PostgresConfig
	NATS     NATSConfig
}

func MustGetCoordinator(envFile string) *CoordinatorConfig {
	cfg := &CoordinatorConfig{}
	if err := cleanenv.ReadConfig(envFile, cfg); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("cfg configure: %v", err)
		}
	}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		log.Fatalf("cfg configure: %v", err)
	}
	if !cfg.Env.IsValid() {
		log.Fatalf("cfg configure: APP_ENV invalid")
	}
	return cfg
}

func MustGetWebhook(envFile string) *WebhookConfig {
	cfg := &WebhookConfig{}
	if err := cleanenv.ReadConfig(envFile, cfg); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("cfg configure: %v", err)
		}
	}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		log.Fatalf("cfg configure: %v", err)
	}
	var err error
	if cfg.MasterKey, err = decodeBase64Key(cfg.MasterKey); err != nil {
		log.Fatalf("cfg configure: %v", err)
	}
	if !cfg.Env.IsValid() {
		log.Fatalf("cfg configure: APP_ENV invalid")
	}
	return cfg
}
