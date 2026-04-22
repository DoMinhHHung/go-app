package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/user-service/internal/domain/entity"
)

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

type userUpserter interface {
	Upsert(ctx context.Context, user *entity.User) error
}

type UserSyncConsumer struct {
	channel *amqp.Channel
	queue   string
	repo    userUpserter
	logger  *zap.Logger
}

func NewUserSyncConsumer(ch *amqp.Channel, queue string, repo userUpserter, logger *zap.Logger) *UserSyncConsumer {
	return &UserSyncConsumer{channel: ch, queue: queue, repo: repo, logger: logger}
}

func (c *UserSyncConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("user sync consumer: consume: %w", err)
	}

	c.logger.Info("user sync consumer started", zap.String("queue", c.queue))

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("user sync consumer stopped")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				c.logger.Warn("user sync channel closed")
				return nil
			}
			c.processMessage(ctx, msg)
		}
	}
}

func (c *UserSyncConsumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	var event UserSyncEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		c.reject(msg, err, "unmarshal failed")
		return
	}

	if event.EventType != "user_sync" {
		c.logger.Warn("unknown user sync event type, dropping",
			zap.String("event_type", event.EventType),
			zap.String("event_id", event.EventID),
		)
		if ackErr := msg.Ack(false); ackErr != nil {
			c.logger.Error("ack failed", zap.Error(ackErr))
		}
		return
	}

	parsedID, err := uuid.Parse(event.UserID)
	if err != nil {
		c.reject(msg, err, "parse user id failed")
		return
	}

	user := &entity.User{
		ID:           parsedID,
		EmailAddress: event.Email,
		PhoneNumber:  event.PhoneNumber,
		FullName:     event.FullName,
		Role:         entity.Role(event.Role),
		Status:       entity.Status(event.Status),
		CreatedAt:    event.CreatedAt,
		UpdatedAt:    event.UpdatedAt,
	}

	if err := c.repo.Upsert(ctx, user); err != nil {
		c.reject(msg, err, "upsert failed")
		return
	}

	if ackErr := msg.Ack(false); ackErr != nil {
		c.logger.Error("ack failed", zap.Error(ackErr))
		return
	}

	c.logger.Info("user synced", zap.String("user_id", event.UserID), zap.String("event_id", event.EventID))
}

func (c *UserSyncConsumer) reject(msg amqp.Delivery, err error, reason string) {
	requeue := !msg.Redelivered
	if nackErr := msg.Nack(false, requeue); nackErr != nil {
		c.logger.Error("nack failed", zap.Error(nackErr))
	}

	if requeue {
		c.logger.Warn(reason+", requeuing once", zap.String("message_id", msg.MessageId), zap.Error(err))
	} else {
		c.logger.Error(reason+", dropping", zap.String("message_id", msg.MessageId), zap.Error(err))
	}
}
