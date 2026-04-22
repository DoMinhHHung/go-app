package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type rateLimitRepo struct {
	client *redis.Client
}

func NewRateLimitRepository(client *redis.Client) *rateLimitRepo {
	return &rateLimitRepo{client: client}
}

func (r *rateLimitRepo) IncrBy(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}
