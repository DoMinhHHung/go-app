package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/api-gateway/internal/dto"
	"github.com/DoMinhHHung/go-app/api-gateway/internal/handler"
	mw "github.com/DoMinhHHung/go-app/api-gateway/internal/middleware"
	"github.com/DoMinhHHung/go-app/api-gateway/internal/proxy"
)

type Config struct {
	AppName            string
	AppEnv             string
	JWTSecret          string
	RateLimitIPMax     int
	RateLimitIPTTL     time.Duration
	RateLimitDevice    int
	RateLimitDeviceTTL time.Duration
	IdentityServiceURL string
	UserServiceURL     string
}

func New(cfg Config, redisClient *redis.Client, logger *zap.Logger) (*gin.Engine, error) {
	identityProxy, err := proxy.New(cfg.IdentityServiceURL, logger)
	if err != nil {
		return nil, err
	}

	userProxy, err := proxy.New(cfg.UserServiceURL, logger)
	if err != nil {
		return nil, err
	}

	gw := handler.New(identityProxy, userProxy)

	r := gin.New()

	// Global middleware
	r.Use(
		mw.RequestID(),
		mw.Logger(logger),
		gin.Recovery(),
		mw.RateLimitByIP(redisClient, cfg.RateLimitIPMax, cfg.RateLimitIPTTL),
		mw.RateLimitByDevice(redisClient, cfg.RateLimitDevice, cfg.RateLimitDeviceTTL),
	)

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", gw.Health(dto.HealthResponse{
		Status:  "ok",
		Service: cfg.AppName,
		Env:     cfg.AppEnv,
	}))

	v1 := r.Group("/api/v1")

	// ── Public auth routes ────────────────────────────────────────────────
	auth := v1.Group("/auth")
	{
		auth.POST("/register", gw.Register)
		auth.POST("/login", gw.Login)
		auth.POST("/verify-otp", gw.VerifyOTP)
		auth.POST("/resend-otp", gw.ResendOTP)
		auth.POST("/refresh", gw.RefreshToken)
		auth.POST("/logout", gw.Logout)
	}

	// ── Protected routes (JWT required) ──────────────────────────────────
	protected := v1.Group("")
	protected.Use(mw.JWTAuth(cfg.JWTSecret))
	{
		users := protected.Group("/users")
		users.GET("/me", gw.GetMe)
	}

	return r, nil
}
