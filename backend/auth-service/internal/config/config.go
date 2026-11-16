package config

import (
	"errors"
	"os"
	"time"
)

const (
	devAccessSecret  = "dev-access-secret-do-not-use-in-production"
	devRefreshSecret = "dev-refresh-secret-do-not-use-in-production"
)

type Config struct {
	Env              string
	GRPCPort         string
	DatabaseURL      string
	RedisAddr        string
	LogLevel         string
	JWTAccessSecret  string
	JWTRefreshSecret string
	AccessTTL        time.Duration
	RefreshTTL       time.Duration
	RabbitURL        string
}

func Load() Config {
	env := get("APP_ENV", "development")
	defaultLevel := "debug"
	if env == "production" {
		defaultLevel = "info"
	}

	return Config{
		Env:              env,
		GRPCPort:         get("GRPC_PORT", "50051"),
		DatabaseURL:      get("DATABASE_URL", "postgres://auth_user:auth_pass@localhost:5433/auth_db?sslmode=disable"),
		RedisAddr:        get("REDIS_ADDR", "localhost:6379"),
		LogLevel:         get("LOG_LEVEL", defaultLevel),
		JWTAccessSecret:  get("JWT_ACCESS_SECRET", devAccessSecret),
		JWTRefreshSecret: get("JWT_REFRESH_SECRET", devRefreshSecret),
		AccessTTL:        15 * time.Minute,
		RefreshTTL:       7 * 24 * time.Hour,
		RabbitURL:        get("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
	}
}

func (c Config) Validate() error {
	if c.Env == "production" && (c.JWTAccessSecret == devAccessSecret || c.JWTRefreshSecret == devRefreshSecret) {
		return errors.New("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET must be set in production")
	}
	return nil
}

func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}