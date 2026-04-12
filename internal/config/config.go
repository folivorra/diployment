package config

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env string `env:"APP_ENV" env-default:"local"` // local, dev

	HTTP     HTTPConfig
	Postgres PostgresConfig
	Auth     AuthConfig
	GitHub   GitHubConfig
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

var (
	cfg  *Config
	once sync.Once
)

// get читает конфиг один раз и возвращает его.
func get() (*Config, error) {
	var err error
	once.Do(func() {
		cfg = &Config{}
		if err = cleanenv.ReadConfig("config/.env.core", cfg); err != nil {
			err = cleanenv.ReadEnv(cfg)
		}
		if err != nil {
			err = fmt.Errorf("config error: %w", err)
		}
	})
	return cfg, err
}

// MustGet обязательно вернет конфиг или упадет с ошибкой,
// если корректный конфиг невозможно вернуть.
func MustGet() *Config {
	if _, err := get(); err != nil {
		log.Fatalf("cfg configure: %v", err)
	}
	return cfg
}
