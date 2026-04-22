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

	"github.com/DoMinhHHung/go-app/identity-service/internal/config"
	"github.com/DoMinhHHung/go-app/identity-service/internal/delivery/http/handler"
	"github.com/DoMinhHHung/go-app/identity-service/internal/delivery/http/router"
	"github.com/DoMinhHHung/go-app/identity-service/internal/event/publisher"
	"github.com/DoMinhHHung/go-app/identity-service/internal/infra"
	postgresRepo "github.com/DoMinhHHung/go-app/identity-service/internal/repository/postgres"
	redisRepo "github.com/DoMinhHHung/go-app/identity-service/internal/repository/redis"
	"github.com/DoMinhHHung/go-app/identity-service/internal/usecase"
	"github.com/DoMinhHHung/go-app/identity-service/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.AppEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	pgPool, err := infra.NewPostgresPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("postgres init failed", zap.Error(err))
	}
	defer pgPool.Close()
	log.Info("postgres connected")

	redisClient, err := infra.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		log.Fatal("redis init failed", zap.Error(err))
	}
	defer redisClient.Close()
	log.Info("redis connected")

	rabbit, err := infra.NewRabbitMQ(
		cfg.RabbitMQHost, cfg.RabbitMQPort,
		cfg.RabbitMQUsername, cfg.RabbitMQPassword,
		cfg.RabbitMQVHost,
	)
	if err != nil {
		log.Fatal("rabbitmq init failed", zap.Error(err))
	}
	defer rabbit.Close()
	log.Info("rabbitmq connected")

	userRepository := postgresRepo.NewUserRepository(pgPool)
	otpRepository := redisRepo.NewOTPRepository(redisClient)
	rateLimitRepo := redisRepo.NewRateLimitRepository(redisClient)

	notifPublisher, err := publisher.NewNotificationPublisher(
		rabbit.Channel,
		cfg.RabbitMQExchange,
		cfg.RabbitMQRoutingKeyEmail,
	)
	if err != nil {
		log.Fatal("publisher init failed", zap.Error(err))
	}

	authUsecase := usecase.NewAuthUsecase(
		userRepository,
		otpRepository,
		notifPublisher,
		log,
		cfg.OTPTTL,
		cfg.OTPMaxResendPerHour,
	)

	authHandler := handler.NewAuthHandler(authUsecase, log)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := router.New(
		authHandler,
		rateLimitRepo,
		cfg.RateLimitIPMax,
		cfg.RateLimitIPTTL,
		cfg.RateLimitDeviceMax,
		cfg.RateLimitDeviceTTL,
		log,
	)

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("identity-service started", zap.String("port", cfg.AppPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutdown signal", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", zap.Error(err))
		os.Exit(1)
	}

	log.Info("identity-service exited gracefully")
}
