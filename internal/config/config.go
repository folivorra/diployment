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
	Env Env `env:"APP_ENV" env-default:"local"`

	MasterKey []byte `env:"MASTER_KEY" env-required:"true"`

	HTTP     HTTPConfig
	Postgres PostgresConfig
	Auth     AuthConfig
	GitHub   GitHubConfig
	Webhook  WebhookServiceConfig
	NATS     NATSConfig
}

type WebhookConfig struct {
	Env Env `env:"APP_ENV" env-default:"local"`

	MasterKey []byte `env:"MASTER_KEY" env-required:"true"`

	HTTP     HTTPConfig
	Postgres PostgresConfig
	NATS     NATSConfig
}

type CoordinatorConfig struct {
	Env Env `env:"APP_ENV" env-default:"local"`

	Postgres PostgresConfig
	NATS     NATSConfig
	Watchdog WatchdogConfig
}

type WatchdogConfig struct {
	Interval   time.Duration `env:"WATCHDOG_INTERVAL" env-default:"5m"`
	StaleAfter time.Duration `env:"WATCHDOG_STALE_AFTER" env-default:"30m"`
}

type BuilderConfig struct {
	Env Env `env:"APP_ENV" env-default:"local"`

	MasterKey    []byte        `env:"MASTER_KEY" env-required:"true"`
	BuildTimeout time.Duration `env:"BUILD_TIMEOUT" env-default:"30m"`

	NATS  NATSConfig
	MinIO MinIOConfig
}

type DeployerConfig struct {
	Env Env `env:"APP_ENV" env-default:"local"`

	MasterKey      []byte        `env:"MASTER_KEY" env-required:"true"`
	DeployTimeout  time.Duration `env:"DEPLOY_TIMEOUT" env-default:"10m"`
	SSHDialTimeout time.Duration `env:"SSH_DIAL_TIMEOUT" env-default:"30s"`

	NATS  NATSConfig
	MinIO MinIOConfig
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
	key := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
	n, err := base64.StdEncoding.Decode(key, encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding master key: %w", err)
	}
	key = key[:n]
	if n != 16 && n != 24 && n != 32 {
		return nil, fmt.Errorf("master key must be 16, 24, or 32 bytes after base64 decode, got %d", n)
	}
	return key, nil
}

type MinIOConfig struct {
	Endpoint  string `env:"MINIO_ENDPOINT" env-required:"true"`
	AccessKey string `env:"MINIO_ACCESS_KEY" env-required:"true"`
	SecretKey string `env:"MINIO_SECRET_KEY" env-required:"true"`
	UseSSL    bool   `env:"MINIO_USE_SSL" env-default:"false"`
}

func MustGetBuilder(envFile string) *BuilderConfig {
	cfg := &BuilderConfig{}
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

func MustGetDeployer(envFile string) *DeployerConfig {
	cfg := &DeployerConfig{}
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
