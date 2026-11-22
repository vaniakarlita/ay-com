package config

import (
	"errors"
	"os"
)

type Config struct {
	Env         string
	LogLevel    string
	GRPCPort    string
	DatabaseURL string
	RedisAddr   string
}

func Load() *Config {
	return &Config{
		Env:         getEnv("APP_ENV", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),
		GRPCPort:    getEnv("USER_GRPC_PORT", "50053"),
		DatabaseURL: getEnv("USER_DATABASE_URL", "postgres://user_user:user_pass@localhost:5435/user_db?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
	}
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("USER_DATABASE_URL is required")
	}
	if c.GRPCPort == "" {
		return errors.New("USER_GRPC_PORT is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
