package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var incrWithExpireScript = redis.NewScript(`
    local current = redis.call("INCR", KEYS[1])
    if current == 1 then
        redis.call("EXPIRE", KEYS[1], ARGV[1])
    end
    return current
`)

type rateLimitRepo struct {
	client *redis.Client
}

func NewRateLimitRepository(client *redis.Client) *rateLimitRepo {
	return &rateLimitRepo{client: client}
}

func (r *rateLimitRepo) IncrBy(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return incrWithExpireScript.Run(ctx, r.client, []string{key}, int(ttl.Seconds())).Int64()
}
