package router

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	_ "github.com/DoMinhHHung/go-app/identity-service/docs"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/identity-service/internal/delivery/http/handler"
	mw "github.com/DoMinhHHung/go-app/identity-service/internal/delivery/http/middleware"
	domainRepo "github.com/DoMinhHHung/go-app/identity-service/internal/domain/repository"
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" || name == "" {
				return fld.Name
			}
			return name
		})
	}
}

func New(
	authHandler *handler.AuthHandler,
	rateLimitRepo domainRepo.RateLimitRepository,
	rateLimitIPMax int,
	rateLimitIPTTL time.Duration,
	rateLimitDeviceMax int,
	rateLimitDeviceTTL time.Duration,
	logger *zap.Logger,
) *gin.Engine {
	r := gin.New()

	r.Use(requestIDMiddleware())
	r.Use(structuredLogger(logger))
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "identity-service"})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	rl := mw.NewRateLimitMiddleware(rateLimitRepo, mw.RateLimitConfig{
		IPMax:     rateLimitIPMax,
		IPTTL:     rateLimitIPTTL,
		DeviceMax: rateLimitDeviceMax,
		DeviceTTL: rateLimitDeviceTTL,
	})

	v1 := r.Group("/api/v1")
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/verify-otp", authHandler.VerifyOTP)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/resend-otp",
			rl.ByEmail(5, time.Hour),
			authHandler.ResendOTP,
		)
	}

	return r
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}
		c.Set("request_id", reqID)
		c.Header("X-Request-ID", reqID)
		c.Next()
	}
}

func structuredLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		reqID, _ := c.Get("request_id")
		log.Info("http request",
			zap.String("request_id", reqID.(string)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.String("ip", c.ClientIP()),
		)
	}
}

func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
