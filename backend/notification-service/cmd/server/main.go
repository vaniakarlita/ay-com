package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vaniakarlita/ay-com/backend/notification-service/internal/config"
	"github.com/vaniakarlita/ay-com/backend/notification-service/internal/logger"
	"github.com/vaniakarlita/ay-com/backend/notification-service/internal/smtpsender"
)

const queueName = "email"

type message struct {
	Type string `json:"type"`
	To   string `json:"to"`
	Code string `json:"code"`
}

func compose(m message) (string, string, error) {
	switch m.Type {
	case "verification":
		return "Your AY.com verification code",
			fmt.Sprintf("Your verification code is %s.\r\nIt expires in 5 minutes.\r\nIf you did not create an AY.com account, you can ignore this email.", m.Code),
			nil
	case "welcome":
		return "Welcome to AY.com",
			"Your registration was successful. Your account is now active and you can sign in at any time.\r\nWelcome aboard!",
			nil
	default:
		return "", "", fmt.Errorf("unknown message type %q", m.Type)
	}
}

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	log := logger.New(cfg.Env, cfg.LogLevel)
	slog.SetDefault(log)

	sender, err := smtpsender.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.From)
	if err != nil {
		log.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Error("rabbitmq connection failed", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Error("open channel failed", "error", err)
		os.Exit(1)
	}
	defer ch.Close()

	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		log.Error("declare queue failed", "error", err)
		os.Exit(1)
	}
	if err := ch.Qos(1, 0, false); err != nil {
		log.Error("set qos failed", "error", err)
		os.Exit(1)
	}

	deliveries, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		log.Error("consume failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("notification-service listening", "queue", queueName, "smtp", cfg.SMTPHost+":"+cfg.SMTPPort)

	for {
		select {
		case <-ctx.Done():
			log.Info("notification-service stopped")
			return
		case d, ok := <-deliveries:
			if !ok {
				log.Error("rabbitmq channel closed")
				os.Exit(1)
			}

			var m message
			if err := json.Unmarshal(d.Body, &m); err != nil {
				log.Error("malformed message dropped", "error", err)
				_ = d.Nack(false, false)
				continue
			}

			subject, body, err := compose(m)
			if err != nil {
				log.Error("unsupported message dropped", "error", err)
				_ = d.Nack(false, false)
				continue
			}

			if err := sender.Send(m.To, subject, body); err != nil {
				log.Error("send email failed", "type", m.Type, "redelivered", d.Redelivered, "error", err)
				_ = d.Nack(false, !d.Redelivered)
				continue
			}

			log.Info("email sent", "type", m.Type)
			_ = d.Ack(false)
		}
	}
}