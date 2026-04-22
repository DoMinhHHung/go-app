package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/notification-service/internal/email"
)

// EmailEvent mirrors the event published by identity-service's NotificationPublisher.
type EmailEvent struct {
	EventID   string         `json:"event_id"`
	EventType string         `json:"event_type"`
	Recipient string         `json:"recipient"`
	Data      map[string]any `json:"data"`
	CreatedAt time.Time      `json:"created_at"`
}

// EmailHandler routes incoming email events to the correct sender logic.
type EmailHandler struct {
	sender email.Sender
	logger *zap.Logger
}

func NewEmailHandler(sender email.Sender, logger *zap.Logger) *EmailHandler {
	return &EmailHandler{sender: sender, logger: logger}
}

func (h *EmailHandler) Handle(ctx context.Context, body []byte) error {
	var event EmailEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("handler: unmarshal: %w", err)
	}

	h.logger.Info("handling email event",
		zap.String("event_id", event.EventID),
		zap.String("event_type", event.EventType),
		zap.String("recipient", event.Recipient),
	)

	switch event.EventType {
	case "otp_register":
		return h.handleOTPRegister(ctx, event)
	default:
		// Unknown type — log and drop (don't requeue).
		h.logger.Warn("unknown event type, dropping message",
			zap.String("event_type", event.EventType),
			zap.String("event_id", event.EventID),
		)
		return nil
	}
}

func (h *EmailHandler) handleOTPRegister(_ context.Context, event EmailEvent) error {
	otpCode, ok := event.Data["otp_code"].(string)
	if !ok || otpCode == "" {
		return fmt.Errorf("handler: otp_register: missing otp_code in event %s", event.EventID)
	}

	expiresIn := 0
	if v, ok := event.Data["expires_in"]; ok {
		switch val := v.(type) {
		case float64:
			expiresIn = int(val) // JSON numbers unmarshal as float64
		case int:
			expiresIn = val
		}
	}

	if err := h.sender.SendOTPEmail(event.Recipient, otpCode, expiresIn); err != nil {
		h.logger.Error("otp_register: send email failed",
			zap.String("event_id", event.EventID),
			zap.String("recipient", event.Recipient),
			zap.Error(err),
		)
		return err
	}

	h.logger.Info("otp email sent",
		zap.String("event_id", event.EventID),
		zap.String("recipient", event.Recipient),
	)
	return nil
}
