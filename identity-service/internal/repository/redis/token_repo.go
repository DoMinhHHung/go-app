package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrTokenNotFound = errors.New("token: not found or expired")

type tokenRepo struct {
	client *redis.Client
}

func NewTokenRepository(client *redis.Client) *tokenRepo {
	return &tokenRepo{client: client}
}

func (r *tokenRepo) SaveRefreshToken(ctx context.Context, userID, token string, ttl time.Duration) error {
	key := fmt.Sprintf("refresh_token:%s", userID)
	return r.client.Set(ctx, key, token, ttl).Err()
}

func (r *tokenRepo) GetRefreshToken(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("refresh_token:%s", userID)
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrTokenNotFound
	}
	return val, err
}

func (r *tokenRepo) DeleteRefreshToken(ctx context.Context, userID string) error {
	key := fmt.Sprintf("refresh_token:%s", userID)
	return r.client.Del(ctx, key).Err()
}
