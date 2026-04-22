package repository

import (
	"context"
	"time"
)

type OTPRepository interface {
	Save(ctx context.Context, email, code string, ttl time.Duration) error
	Get(ctx context.Context, email string) (string, error)
	Delete(ctx context.Context, email string) error
	IncrResendCount(ctx context.Context, email string, windowTTL time.Duration) (int64, error)
	GetResendCount(ctx context.Context, email string) (int64, error)
	IncrAttemptCount(ctx context.Context, email string, ttl time.Duration) (int64, error)
	DeleteAttemptCount(ctx context.Context, email string) error
	GetAttemptCount(ctx context.Context, email string) (int64, error)
}

type RateLimitRepository interface {
	IncrBy(ctx context.Context, key string, ttl time.Duration) (int64, error)
}
