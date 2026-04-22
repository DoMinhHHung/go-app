package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// incrWithTTL atomically increments a counter, setting TTL only on first creation.
var incrWithTTL = redis.NewScript(`
	local current = redis.call("INCR", KEYS[1])
	if current == 1 then
		redis.call("EXPIRE", KEYS[1], ARGV[1])
	end
	return current
`)

func rateLimitError(c *gin.Context, code string) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"success": false,
		"message": "too many requests, please slow down",
		"code":    code,
	})
}

// RateLimitByIP limits requests per client IP across the entire gateway.
func RateLimitByIP(client *redis.Client, max int, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("gw:rl:ip:%s", c.ClientIP())
		count, err := runIncrScript(c.Request.Context(), client, key, ttl)
		if err != nil {
			// Fail open: if Redis is down, let the request through.
			c.Next()
			return
		}
		if count > int64(max) {
			rateLimitError(c, "RATE_LIMIT_IP")
			return
		}
		c.Next()
	}
}

// RateLimitByDevice limits requests per device fingerprint (optional header).
func RateLimitByDevice(client *redis.Client, max int, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		fingerprint := c.GetHeader("X-Device-Fingerprint")
		if fingerprint == "" {
			c.Next()
			return
		}
		key := fmt.Sprintf("gw:rl:device:%s", fingerprint)
		count, err := runIncrScript(c.Request.Context(), client, key, ttl)
		if err != nil {
			c.Next()
			return
		}
		if count > int64(max) {
			rateLimitError(c, "RATE_LIMIT_DEVICE")
			return
		}
		c.Next()
	}
}

func runIncrScript(ctx context.Context, client *redis.Client, key string, ttl time.Duration) (int64, error) {
	return incrWithTTL.Run(ctx, client, []string{key}, int(ttl.Seconds())).Int64()
}
