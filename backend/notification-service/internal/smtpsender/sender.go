package smtpsender

import (
	"fmt"
	"net/mail"
	"net/smtp"
)

type Sender struct {
	host string
	port string
	user string
	pass string
	from mail.Address
}

func New(host, port, user, pass, from string) (*Sender, error) {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_FROM: %w", err)
	}
	return &Sender{host: host, port: port, user: user, pass: pass, from: *addr}, nil
}

func (s *Sender) Send(to, subject, body string) error {
	rcpt, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from.String(), rcpt.Address, subject, body,
	)

	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}
	return smtp.SendMail(s.host+":"+s.port, auth, s.from.Address, []string{rcpt.Address}, []byte(msg))
}