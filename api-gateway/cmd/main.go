package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/api-gateway/internal/config"
	"github.com/DoMinhHHung/go-app/api-gateway/internal/infra"
	"github.com/DoMinhHHung/go-app/api-gateway/internal/router"
	"github.com/DoMinhHHung/go-app/api-gateway/pkg/logger"
)

// @title           API Gateway
// @version         1.0
// @description     Cổng vào duy nhất cho tất cả microservices. Xử lý authentication, rate limiting, và routing.
// @host            localhost:8000
// @BasePath        /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Nhập "Bearer " và theo sau là JWT access token. Ví dụ: "Bearer eyJhbGci..."
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config error: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.AppEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger error: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	redisClient, err := infra.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		log.Fatal("redis connect failed", zap.Error(err))
	}
	defer redisClient.Close()
	log.Info("redis connected")

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r, err := router.New(router.Config{
		AppName:            cfg.AppName,
		AppEnv:             cfg.AppEnv,
		JWTSecret:          cfg.JWTAccessSecret,
		RateLimitIPMax:     cfg.RateLimitIPMax,
		RateLimitIPTTL:     cfg.RateLimitIPTTL,
		RateLimitDevice:    cfg.RateLimitDeviceMax,
		RateLimitDeviceTTL: cfg.RateLimitDeviceTTL,
		IdentityServiceURL: cfg.IdentityServiceURL,
		UserServiceURL:     cfg.UserServiceURL,
	}, redisClient, log)
	if err != nil {
		log.Fatal("router init failed", zap.Error(err))
	}

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("api-gateway started",
			zap.String("port", cfg.AppPort),
			zap.String("env", cfg.AppEnv),
			zap.String("identity_url", cfg.IdentityServiceURL),
			zap.String("user_url", cfg.UserServiceURL),
			zap.String("swagger", "http://localhost:"+cfg.AppPort+"/swagger/index.html"),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", zap.Error(err))
		os.Exit(1)
	}

	log.Info("api-gateway exited gracefully")
}
