package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrOTPNotFound = errors.New("otp: not found or expired")

var incrWithTTL = redis.NewScript(`
	local current = redis.call("INCR", KEYS[1])
	if current == 1 then
		redis.call("EXPIRE", KEYS[1], ARGV[1])
	end
	return current
`)

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

func (r *otpRepo) GetAttemptCount(ctx context.Context, email string) (int64, error) {
	key := fmt.Sprintf("otp:attempts:%s", email)
	val, err := r.client.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return val, err
}

func (r *otpRepo) IncrResendCount(ctx context.Context, email string, windowTTL time.Duration) (int64, error) {
	key := fmt.Sprintf("otp:resend_count:%s", email)
	res, err := incrWithTTL.Run(ctx, r.client, []string{key}, int(windowTTL.Seconds())).Int64()
	if err != nil {
		return 0, err
	}
	return res, nil
}

func (r *otpRepo) GetResendCount(ctx context.Context, email string) (int64, error) {
	key := fmt.Sprintf("otp:resend_count:%s", email)
	val, err := r.client.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return val, err
}

func (r *otpRepo) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	ok, err := r.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis: acquire lock %q: %w", key, err)
	}
	return ok, nil
}

func (r *otpRepo) ReleaseLock(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *otpRepo) IncrAttemptCount(ctx context.Context, email string, ttl time.Duration) (int64, error) {
	key := fmt.Sprintf("otp:attempts:%s", email)
	res, err := incrWithTTL.Run(ctx, r.client, []string{key}, int(ttl.Seconds())).Int64()
	if err != nil {
		return 0, err
	}
	return res, nil
}

func (r *otpRepo) DeleteAttemptCount(ctx context.Context, email string) error {
	key := fmt.Sprintf("otp:attempts:%s", email)
	return r.client.Del(ctx, key).Err()
}
