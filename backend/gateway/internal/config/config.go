package config

import "os"

type Config struct {
	Env           string
	Port          string
	AuthAddr      string
	AllowedOrigin string
	LogLevel      string
}

func Load() Config {
	env := get("APP_ENV", "development")
	defaultLevel := "debug"
	if env == "production" {
		defaultLevel = "info"
	}

	return Config{
		Env:           env,
		Port:          get("PORT", "8080"),
		AuthAddr:      get("AUTH_SERVICE_ADDR", "localhost:50051"),
		AllowedOrigin: get("ALLOWED_ORIGIN", "http://localhost:5173"),
		LogLevel:      get("LOG_LEVEL", defaultLevel),
	}
}

func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}