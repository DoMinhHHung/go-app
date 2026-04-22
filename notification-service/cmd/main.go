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
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/notification-service/internal/config"
	"github.com/DoMinhHHung/go-app/notification-service/internal/consumer"
	"github.com/DoMinhHHung/go-app/notification-service/internal/email"
	"github.com/DoMinhHHung/go-app/notification-service/internal/handler"
	"github.com/DoMinhHHung/go-app/notification-service/internal/infra"
	"github.com/DoMinhHHung/go-app/notification-service/pkg/logger"

	_ "github.com/DoMinhHHung/go-app/notification-service/docs"
)

// @title           Notification Service API
// @version         1.0
// @description     Service nhận message từ RabbitMQ và gửi email thông báo (OTP, v.v.).
// @host            localhost:8082
// @BasePath        /
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

	rabbit, err := infra.NewRabbitMQ(
		cfg.RabbitMQHost, cfg.RabbitMQPort,
		cfg.RabbitMQUsername, cfg.RabbitMQPassword,
		cfg.RabbitMQVHost,
	)
	if err != nil {
		log.Fatal("rabbitmq connect failed", zap.Error(err))
	}
	defer rabbit.Close()
	log.Info("rabbitmq connected")

	if err := rabbit.SetupConsumerQueue(cfg.RabbitMQExchange, cfg.RabbitMQQueue, cfg.RabbitMQRoutingKey); err != nil {
		log.Fatal("rabbitmq setup failed", zap.Error(err))
	}
	log.Info("rabbitmq queue bound",
		zap.String("exchange", cfg.RabbitMQExchange),
		zap.String("queue", cfg.RabbitMQQueue),
		zap.String("routing_key", cfg.RabbitMQRoutingKey),
	)

	emailSender := email.NewSMTPSender(
		cfg.SMTPHost, cfg.SMTPPort,
		cfg.SMTPUsername, cfg.SMTPPassword,
		cfg.SMTPFrom,
	)

	emailHandler := handler.NewEmailHandler(emailSender, log)
	emailConsumer := consumer.NewEmailConsumer(rabbit.Channel, cfg.RabbitMQQueue, emailHandler, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- emailConsumer.Start(ctx)
	}()

	// HTTP server for health check + swagger
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/health", healthHandler(cfg.AppName))

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("notification-service started",
			zap.String("port", cfg.AppPort),
			zap.String("queue", cfg.RabbitMQQueue),
			zap.String("swagger", "http://localhost:"+cfg.AppPort+"/swagger/index.html"),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("health server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info("shutdown signal received", zap.String("signal", sig.String()))
	case err := <-consumerDone:
		if err != nil {
			log.Error("consumer stopped with error", zap.Error(err))
		}
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced shutdown", zap.Error(err))
		os.Exit(1)
	}

	log.Info("notification-service exited gracefully")
}

// healthHandler godoc
// @Summary      Health check
// @Description  Kiểm tra notification-service có đang chạy không
// @Tags         system
// @Produce      json
// @Success      200 {object} map[string]string
// @Router       /health [get]
func healthHandler(appName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": appName})
	}
}
