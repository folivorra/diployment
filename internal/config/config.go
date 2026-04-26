package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env string `env:"APP_ENV" env-default:"local"` // local, dev

	MasterKey string `env:"MASTER_KEY" env-required:"true"`

	HTTP     HTTPConfig
	Postgres PostgresConfig
	Auth     AuthConfig
	GitHub   GitHubConfig
	Webhook  WebhookConfig
}

type WebhookConfig struct {
	URL string `env:"WEBHOOK_URL" env-required:"true"`
}

type HTTPConfig struct {
	Port string `env:"HTTP_PORT" env-default:"8080"`
	Host string `env:"HTTP_HOST" env-default:"localhost"`
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

var (
	cfg  *Config
	once sync.Once
)

func (h HTTPConfig) Address() string {
	return net.JoinHostPort(h.Host, h.Port)
}

func decodeBase64Key(encoded string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decoding master key: %w", err)
	}
	return string(key), nil
}

// get читает конфиг один раз и возвращает его.
func get() (*Config, error) {
	var err error
	once.Do(func() {
		cfg = &Config{}
		if err = cleanenv.ReadConfig("config/.core.env", cfg); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return
			}
			err = nil
		}
		if err = cleanenv.ReadEnv(cfg); err != nil {
			return
		}
		if cfg.MasterKey, err = decodeBase64Key(cfg.MasterKey); err != nil {
			return
		}
		switch cfg.Env {
		case "local", "dev":
		default:
			err = fmt.Errorf("unknown environment: %s", cfg.Env)
		}
	})
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// MustGet обязательно вернет конфиг или упадет с ошибкой, если корректный конфиг невозможно вернуть.
func MustGet() *Config {
	if _, err := get(); err != nil {
		log.Fatalf("cfg configure: %v", err)
	}
	return cfg
}
