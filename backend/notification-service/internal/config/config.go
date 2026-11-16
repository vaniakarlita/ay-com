package config

import "os"

type Config struct {
	Env       string
	LogLevel  string
	RabbitURL string
	SMTPHost  string
	SMTPPort  string
	SMTPUser  string
	SMTPPass  string
	From      string
}

func Load() Config {
	env := get("APP_ENV", "development")
	defaultLevel := "debug"
	if env == "production" {
		defaultLevel = "info"
	}

	return Config{
		Env:       env,
		LogLevel:  get("LOG_LEVEL", defaultLevel),
		RabbitURL: get("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		SMTPHost:  get("SMTP_HOST", "localhost"),
		SMTPPort:  get("SMTP_PORT", "1025"),
		SMTPUser:  get("SMTP_USER", ""),
		SMTPPass:  get("SMTP_PASS", ""),
		From:      get("SMTP_FROM", "AY.com <no-reply@ay.com>"),
	}
}

func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}