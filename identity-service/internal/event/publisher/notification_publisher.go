package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EmailEvent struct {
	EventID   string         `json:"event_id"`
	EventType string         `json:"event_type"`
	Recipient string         `json:"recipient"`
	Data      map[string]any `json:"data"`
	CreatedAt time.Time      `json:"created_at"`
}

type NotificationPublisher struct {
	channel    *amqp.Channel
	exchange   string
	routingKey string
}

func NewNotificationPublisher(ch *amqp.Channel, exchange, routingKey string) (*NotificationPublisher, error) {
	err := ch.ExchangeDeclare(
		exchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("publisher: declare exchange: %w", err)
	}

	return &NotificationPublisher{
		channel:    ch,
		exchange:   exchange,
		routingKey: routingKey,
	}, nil
}

func (p *NotificationPublisher) PublishOTPEmail(ctx context.Context, eventID, recipient, otpCode string, expireSeconds int) error {
	event := EmailEvent{
		EventID:   eventID,
		EventType: "otp_register",
		Recipient: recipient,
		Data: map[string]any{
			"otp_code":   otpCode,
			"expires_in": expireSeconds,
		},
		CreatedAt: time.Now().UTC(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("publisher: marshal: %w", err)
	}

	return p.channel.PublishWithContext(ctx,
		p.exchange,
		p.routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    eventID,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
}
