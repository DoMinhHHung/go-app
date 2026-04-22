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
	"github.com/DoMinhHHung/go-app/identity-service/pkg/logger"
)

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

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": cfg.AppName,
			"env":     cfg.AppEnv,
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server starting",
			zap.String("service", cfg.AppName),
			zap.String("port", cfg.AppPort),
			zap.String("env", cfg.AppEnv),
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

	log.Info("server exited gracefully")
}
