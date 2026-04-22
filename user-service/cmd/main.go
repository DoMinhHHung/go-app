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

	"github.com/DoMinhHHung/go-app/user-service/internal/config"
	"github.com/DoMinhHHung/go-app/user-service/pkg/logger"

	_ "github.com/DoMinhHHung/go-app/user-service/docs"
)

// @title           User Service API
// @version         1.0
// @description     Quản lý thông tin profile người dùng.
// @host            localhost:8081
// @BasePath        /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Nhập "Bearer " và theo sau là JWT access token.
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

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/health", healthHandler(cfg.AppName))

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("user-service started",
			zap.String("port", cfg.AppPort),
			zap.String("swagger", "http://localhost:"+cfg.AppPort+"/swagger/index.html"),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server listen error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced shutdown", zap.Error(err))
		os.Exit(1)
	}

	log.Info("user-service exited gracefully")
}

// healthHandler godoc
// @Summary      Health check
// @Description  Kiểm tra user-service có đang chạy không
// @Tags         system
// @Produce      json
// @Success      200 {object} map[string]string
// @Router       /health [get]
func healthHandler(appName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": appName})
	}
}
