package consumer

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/notification-service/internal/handler"
)

type EmailConsumer struct {
	channel *amqp.Channel
	queue   string
	handler *handler.EmailHandler
	logger  *zap.Logger
}

func NewEmailConsumer(
	ch *amqp.Channel,
	queue string,
	h *handler.EmailHandler,
	logger *zap.Logger,
) *EmailConsumer {
	return &EmailConsumer{
		channel: ch,
		queue:   queue,
		handler: h,
		logger:  logger,
	}
}

func (c *EmailConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	c.logger.Info("email consumer started", zap.String("queue", c.queue))

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("email consumer stopped")
			return nil

		case msg, ok := <-msgs:
			if !ok {
				c.logger.Warn("rabbitmq channel closed")
				return nil
			}
			c.processMessage(ctx, msg)
		}
	}
}

func (c *EmailConsumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	err := c.handler.Handle(ctx, msg.Body)
	if err == nil {
		// Happy path: acknowledge so RabbitMQ removes it from the queue.
		if ackErr := msg.Ack(false); ackErr != nil {
			c.logger.Error("ack failed", zap.Error(ackErr))
		}
		return
	}

	requeue := !msg.Redelivered
	if nackErr := msg.Nack(false, requeue); nackErr != nil {
		c.logger.Error("nack failed", zap.Error(nackErr))
	}

	if requeue {
		c.logger.Warn("message processing failed, requeuing once",
			zap.String("message_id", msg.MessageId),
			zap.Error(err),
		)
	} else {
		c.logger.Error("message processing failed after retry, dropping",
			zap.String("message_id", msg.MessageId),
			zap.Error(err),
		)
	}
}
