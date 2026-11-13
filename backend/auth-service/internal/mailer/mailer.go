package mailer

import (
	"context"
	"log/slog"
)

type LogMailer struct {
	log *slog.Logger
}

func NewLogMailer(log *slog.Logger) *LogMailer {
	return &LogMailer{log: log}
}

func (m *LogMailer) SendVerification(_ context.Context, to, code string) error {
	m.log.Debug("verification email (dev only)", "to", to, "code", code)
	return nil
}