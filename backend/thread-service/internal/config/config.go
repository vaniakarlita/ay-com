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
		GRPCPort:    getEnv("THREAD_GRPC_PORT", "50052"),
		DatabaseURL: getEnv("THREAD_DATABASE_URL", "postgres://thread_user:thread_pass@localhost:5434/thread_db?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
	}
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("THREAD_DATABASE_URL is required")
	}
	if c.GRPCPort == "" {
		return errors.New("THREAD_GRPC_PORT is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
