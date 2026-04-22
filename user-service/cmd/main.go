package main

// @title           User Service API
// @version         1.0
// @description     Quản lý thông tin profile người dùng. Tất cả routes yêu cầu JWT được validate bởi API Gateway.
// @host            localhost:8081
// @BasePath        /
//
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Nhập "Bearer " và JWT access token. VD: "Bearer eyJhbGci..."

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

	"github.com/DoMinhHHung/go-app/user-service/internal/config"
	"github.com/DoMinhHHung/go-app/user-service/internal/consumer"
	"github.com/DoMinhHHung/go-app/user-service/internal/delivery/http/handler"
	"github.com/DoMinhHHung/go-app/user-service/internal/delivery/http/router"
	"github.com/DoMinhHHung/go-app/user-service/internal/infra"
	postgresRepo "github.com/DoMinhHHung/go-app/user-service/internal/repository/postgres"
	"github.com/DoMinhHHung/go-app/user-service/internal/usecase"
	"github.com/DoMinhHHung/go-app/user-service/pkg/logger"
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

	if err := infra.RunMigrations(context.Background(), pgPool, log); err != nil {
		log.Fatal("migration failed", zap.Error(err))
	}
	log.Info("database schema up to date")

	rabbit, err := infra.NewRabbitMQ(cfg.RabbitMQHost, cfg.RabbitMQPort, cfg.RabbitMQUsername, cfg.RabbitMQPassword, cfg.RabbitMQVHost)
	if err != nil {
		log.Fatal("rabbitmq init failed", zap.Error(err))
	}
	defer rabbit.Close()
	log.Info("rabbitmq connected")

	if err := rabbit.SetupConsumerQueue(cfg.RabbitMQExchange, cfg.RabbitMQQueue, cfg.RabbitMQRoutingKey); err != nil {
		log.Fatal("rabbitmq queue setup failed", zap.Error(err))
	}
	log.Info("rabbitmq queue bound",
		zap.String("exchange", cfg.RabbitMQExchange),
		zap.String("queue", cfg.RabbitMQQueue),
		zap.String("routing_key", cfg.RabbitMQRoutingKey),
	)

	userRepository := postgresRepo.NewUserRepository(pgPool)
	userUsecase := usecase.NewUserUsecase(userRepository, log)
	userHandler := handler.NewUserHandler(userUsecase, log)
	userSyncConsumer := consumer.NewUserSyncConsumer(rabbit.Channel, cfg.RabbitMQQueue, userRepository, log)
	appCtx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	go func() {
		if err := userSyncConsumer.Start(appCtx); err != nil {
			log.Error("user sync consumer stopped with error", zap.Error(err))
		}
	}()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := router.New(userHandler, log)

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("user-service started", zap.String("port", cfg.AppPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutdown signal", zap.String("signal", sig.String()))
	cancelApp()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", zap.Error(err))
		os.Exit(1)
	}

	log.Info("user-service exited gracefully")
}
