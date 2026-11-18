package mailer

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const QueueName = "email"

type Message struct {
	Type string `json:"type"`
	To   string `json:"to"`
	Code string `json:"code,omitempty"`
}

type RabbitMailer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	mu   sync.Mutex
}

func NewRabbitMailer(url string) (*RabbitMailer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := ch.QueueDeclare(QueueName, true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, err
	}
	return &RabbitMailer{conn: conn, ch: ch}, nil
}

func (m *RabbitMailer) publish(ctx context.Context, msg Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ch.PublishWithContext(ctx, "", QueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Body:         body,
	})
}

func (m *RabbitMailer) SendVerification(ctx context.Context, to, code string) error {
	return m.publish(ctx, Message{Type: "verification", To: to, Code: code})
}

func (m *RabbitMailer) SendWelcome(ctx context.Context, to string) error {
	return m.publish(ctx, Message{Type: "welcome", To: to})
}

func (m *RabbitMailer) Close() {
	m.ch.Close()
	m.conn.Close()
}