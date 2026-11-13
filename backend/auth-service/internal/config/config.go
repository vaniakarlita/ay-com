package config

import "os"

type Config struct {
	Env         string
	GRPCPort    string
	DatabaseURL string
	RedisAddr   string
	LogLevel    string
}

func Load() Config {
	env := get("APP_ENV", "development")
	defaultLevel := "debug"
	if env == "production" {
		defaultLevel = "info"
	}

	return Config{
		Env:         env,
		GRPCPort:    get("GRPC_PORT", "50051"),
		DatabaseURL: get("DATABASE_URL", "postgres://auth_user:auth_pass@localhost:5433/auth_db?sslmode=disable"),
		RedisAddr:   get("REDIS_ADDR", "localhost:6379"),
		LogLevel:    get("LOG_LEVEL", defaultLevel),
	}
}

func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}