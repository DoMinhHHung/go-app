package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DoMinhHHung/go-app/identity-service/internal/delivery/http/dto"
	domainRepo "github.com/DoMinhHHung/go-app/identity-service/internal/domain/repository"
)

type RateLimitConfig struct {
	IPMax     int
	IPTTL     time.Duration
	DeviceMax int
	DeviceTTL time.Duration
}

type RateLimitMiddleware struct {
	repo domainRepo.RateLimitRepository
	cfg  RateLimitConfig
}

func NewRateLimitMiddleware(repo domainRepo.RateLimitRepository, cfg RateLimitConfig) *RateLimitMiddleware {
	return &RateLimitMiddleware{repo: repo, cfg: cfg}
}

func (m *RateLimitMiddleware) ByIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("rate_limit:ip:%s", ip)

		count, err := m.repo.IncrBy(c.Request.Context(), key, m.cfg.IPTTL)
		if err != nil {
			c.Next()
			return
		}

		if count > int64(m.cfg.IPMax) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				dto.Fail("too many requests from your IP", "RATE_LIMIT_IP"))
			return
		}

		c.Next()
	}
}

func (m *RateLimitMiddleware) ByDevice() gin.HandlerFunc {
	return func(c *gin.Context) {
		fingerprint := c.GetHeader("X-Device-Fingerprint")
		if fingerprint == "" {
			c.Next()
			return
		}

		key := fmt.Sprintf("rate_limit:device:%s", fingerprint)
		count, err := m.repo.IncrBy(c.Request.Context(), key, m.cfg.DeviceTTL)
		if err != nil {
			c.Next()
			return
		}

		if count > int64(m.cfg.DeviceMax) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				dto.Fail("too many requests from your device", "RATE_LIMIT_DEVICE"))
			return
		}

		c.Next()
	}
}

func (m *RateLimitMiddleware) ByEmail(max int, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			EmailAddress string `json:"email_address"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.EmailAddress == "" {
			c.Next()
			return
		}

		key := fmt.Sprintf("rate_limit:email:%s", body.EmailAddress)
		count, err := m.repo.IncrBy(c.Request.Context(), key, ttl)
		if err != nil {
			c.Next()
			return
		}

		if count > int64(max) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				dto.Fail("too many OTP requests for this email", "RATE_LIMIT_EMAIL"))
			return
		}

		c.Set("email_from_body", body.EmailAddress)
		c.Next()
	}
}
