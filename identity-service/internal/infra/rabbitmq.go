package infra

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewRabbitMQ(host string, port int, username, password, vhost string) (*RabbitMQ, error) {
	dsn := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", username, password, host, port, vhost)

	conn, err := amqp.Dial(dsn)
	if err != nil {
		safeURL := fmt.Sprintf("amqp://%s:***@%s:%d/%s", username, host, port, vhost)
		return nil, fmt.Errorf("rabbitmq: dial %s: %w", safeURL, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq: open channel: %w", err)
	}

	return &RabbitMQ{Conn: conn, Channel: ch}, nil
}

func (r *RabbitMQ) Close() {
	if r.Channel != nil {
		r.Channel.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}
