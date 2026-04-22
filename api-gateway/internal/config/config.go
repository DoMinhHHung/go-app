package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv  string
	AppPort string
	AppName string

	IdentityServiceURL     string
	UserServiceURL         string
	NotificationServiceURL string

	RedisHost     string
	RedisPort     string
	RedisPassword string

	JWTAccessSecret string

	RateLimitIPMax     int
	RateLimitIPTTL     time.Duration
	RateLimitDeviceMax int
	RateLimitDeviceTTL time.Duration
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config: read .env: %w", err)
	}

	rateLimitIPTTL, err := time.ParseDuration(viper.GetString("RATE_LIMIT_IP_TTL"))
	if err != nil {
		return nil, fmt.Errorf("config: RATE_LIMIT_IP_TTL: %w", err)
	}

	rateLimitDeviceTTL, err := time.ParseDuration(viper.GetString("RATE_LIMIT_DEVICE_TTL"))
	if err != nil {
		return nil, fmt.Errorf("config: RATE_LIMIT_DEVICE_TTL: %w", err)
	}

	cfg := &Config{
		AppEnv:  viper.GetString("APP_ENV"),
		AppPort: viper.GetString("APP_PORT"),
		AppName: viper.GetString("APP_NAME"),

		IdentityServiceURL:     viper.GetString("IDENTITY_SERVICE_URL"),
		UserServiceURL:         viper.GetString("USER_SERVICE_URL"),
		NotificationServiceURL: viper.GetString("NOTIFICATION_SERVICE_URL"),

		RedisHost:     viper.GetString("REDIS_HOST"),
		RedisPort:     viper.GetString("REDIS_PORT"),
		RedisPassword: viper.GetString("REDIS_PASSWORD"),

		JWTAccessSecret: viper.GetString("JWT_ACCESS_SECRET"),

		RateLimitIPMax:     viper.GetInt("RATE_LIMIT_IP_MAX"),
		RateLimitIPTTL:     rateLimitIPTTL,
		RateLimitDeviceMax: viper.GetInt("RATE_LIMIT_DEVICE_MAX"),
		RateLimitDeviceTTL: rateLimitDeviceTTL,
	}

	return cfg, validate(cfg)
}

func validate(cfg *Config) error {
	required := []struct {
		name  string
		value string
	}{
		{"IDENTITY_SERVICE_URL", cfg.IdentityServiceURL},
		{"USER_SERVICE_URL", cfg.UserServiceURL},
		{"REDIS_HOST", cfg.RedisHost},
		{"JWT_ACCESS_SECRET", cfg.JWTAccessSecret},
	}

	for _, r := range required {
		if r.value == "" {
			return fmt.Errorf("config: %s is required", r.name)
		}
	}
	return nil
}
