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

	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("rabbitmq: set qos: %w", err)
	}

	return &RabbitMQ{Conn: conn, Channel: ch}, nil
}

func (r *RabbitMQ) SetupConsumerQueue(exchange, queue, routingKey string) error {
	if err := r.Channel.ExchangeDeclare(exchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("rabbitmq: declare exchange: %w", err)
	}

	if _, err := r.Channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("rabbitmq: declare queue: %w", err)
	}

	if err := r.Channel.QueueBind(queue, routingKey, exchange, false, nil); err != nil {
		return fmt.Errorf("rabbitmq: bind queue: %w", err)
	}

	return nil
}

func (r *RabbitMQ) Close() {
	if r.Channel != nil {
		r.Channel.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}
