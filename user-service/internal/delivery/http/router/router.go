package router

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/user-service/internal/delivery/http/handler"
)

func New(userHandler *handler.UserHandler, logger *zap.Logger) *gin.Engine {
	r := gin.New()
	r.Use(requestIDMiddleware(), structuredLogger(logger), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "user-service"})
	})

	v1 := r.Group("/api/v1")
	users := v1.Group("/users")
	{
		users.GET("/me", userHandler.GetMe)
		users.PUT("/me", userHandler.UpdateMe)
		users.DELETE("/me", userHandler.DeleteMe)
		users.GET("/", userHandler.ListUsers)
	}

	return r
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			b := make([]byte, 8)
			rand.Read(b)
			reqID = fmt.Sprintf("%x", b)
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
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.String("ip", c.ClientIP()),
			zap.String("user_id", c.GetHeader("X-User-ID")),
			zap.Duration("latency", time.Since(time.Now())),
		)
	}
}
