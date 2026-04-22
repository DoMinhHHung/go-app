package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrOTPNotFound = errors.New("otp: not found or expired")

type otpRepo struct {
	client *redis.Client
}

func NewOTPRepository(client *redis.Client) *otpRepo {
	return &otpRepo{client: client}
}

func (r *otpRepo) Save(ctx context.Context, email, code string, ttl time.Duration) error {
	key := fmt.Sprintf("otp:%s", email)
	return r.client.Set(ctx, key, code, ttl).Err()
}

func (r *otpRepo) Get(ctx context.Context, email string) (string, error) {
	key := fmt.Sprintf("otp:%s", email)
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrOTPNotFound
	}
	return val, err
}

func (r *otpRepo) Delete(ctx context.Context, email string) error {
	key := fmt.Sprintf("otp:%s", email)
	return r.client.Del(ctx, key).Err()
}

func (r *otpRepo) IncrResendCount(ctx context.Context, email string, windowTTL time.Duration) (int64, error) {
	key := fmt.Sprintf("otp:resend_count:%s", email)
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, windowTTL)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

func (r *otpRepo) GetResendCount(ctx context.Context, email string) (int64, error) {
	key := fmt.Sprintf("otp:resend_count:%s", email)
	val, err := r.client.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return val, err
}
