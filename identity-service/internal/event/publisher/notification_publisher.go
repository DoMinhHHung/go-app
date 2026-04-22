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

type UserSyncEvent struct {
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	PhoneNumber *string   `json:"phone_number"`
	FullName    string    `json:"full_name"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type NotificationPublisher struct {
	channel            *amqp.Channel
	exchange           string
	emailRoutingKey    string
	userSyncRoutingKey string
}

func NewNotificationPublisher(ch *amqp.Channel, exchange, emailRoutingKey, userSyncRoutingKey string) (*NotificationPublisher, error) {
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
		channel:            ch,
		exchange:           exchange,
		emailRoutingKey:    emailRoutingKey,
		userSyncRoutingKey: userSyncRoutingKey,
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

	return p.publish(ctx, p.emailRoutingKey, eventID, body)
}

func (p *NotificationPublisher) PublishUserSync(ctx context.Context, eventID, userID, email string, phoneNumber *string, fullName, role, status string, createdAt, updatedAt time.Time) error {
	event := UserSyncEvent{
		EventID:     eventID,
		EventType:   "user_sync",
		UserID:      userID,
		Email:       email,
		PhoneNumber: phoneNumber,
		FullName:    fullName,
		Role:        role,
		Status:      status,
		CreatedAt:   createdAt.UTC(),
		UpdatedAt:   updatedAt.UTC(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("publisher: marshal: %w", err)
	}

	return p.publish(ctx, p.userSyncRoutingKey, eventID, body)
}

func (p *NotificationPublisher) publish(ctx context.Context, routingKey, eventID string, body []byte) error {
	return p.channel.PublishWithContext(ctx,
		p.exchange,
		routingKey,
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
