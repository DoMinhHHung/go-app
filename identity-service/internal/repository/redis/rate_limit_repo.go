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
	script := `
		local current = redis.call("INCR", KEYS[1])
		if current == 1 then
			redis.call("EXPIRE", KEYS[1], ARGV[1])
		end
		return current
	`
	res, err := r.client.Eval(ctx, script, []string{key}, int(ttl.Seconds())).Int64()
	if err != nil {
		return 0, err
	}
	return res, nil
}
