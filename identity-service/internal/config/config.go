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

	DatabaseURL string

	RedisHost     string
	RedisPort     string
	RedisPassword string

	RabbitMQHost               string
	RabbitMQPort               int
	RabbitMQUsername           string
	RabbitMQPassword           string
	RabbitMQVHost              string
	RabbitMQExchange           string
	RabbitMQRoutingKeyEmail    string
	RabbitMQRoutingKeyUserSync string

	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration

	OTPTTL              time.Duration
	OTPMaxResendPerHour int

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

	jwtAccessTTL, err := time.ParseDuration(viper.GetString("JWT_ACCESS_TTL"))
	if err != nil {
		return nil, fmt.Errorf("config: JWT_ACCESS_TTL: %w", err)
	}

	jwtRefreshTTL, err := time.ParseDuration(viper.GetString("JWT_REFRESH_TTL"))
	if err != nil {
		return nil, fmt.Errorf("config: JWT_REFRESH_TTL: %w", err)
	}

	otpTTL, err := time.ParseDuration(viper.GetString("OTP_TTL"))
	if err != nil {
		return nil, fmt.Errorf("config: OTP_TTL: %w", err)
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

		DatabaseURL: viper.GetString("DATABASE_URL"),

		RedisHost:     viper.GetString("REDIS_HOST"),
		RedisPort:     viper.GetString("REDIS_PORT"),
		RedisPassword: viper.GetString("REDIS_PASSWORD"),

		RabbitMQHost:               viper.GetString("RABBITMQ_HOST"),
		RabbitMQPort:               viper.GetInt("RABBITMQ_PORT"),
		RabbitMQUsername:           viper.GetString("RABBITMQ_USERNAME"),
		RabbitMQPassword:           viper.GetString("RABBITMQ_PASSWORD"),
		RabbitMQVHost:              viper.GetString("RABBITMQ_VHOST"),
		RabbitMQExchange:           viper.GetString("RABBITMQ_EXCHANGE"),
		RabbitMQRoutingKeyEmail:    viper.GetString("RABBITMQ_ROUTING_KEY_EMAIL"),
		RabbitMQRoutingKeyUserSync: viper.GetString("RABBITMQ_ROUTING_KEY_USER_SYNC"),

		JWTAccessSecret:  viper.GetString("JWT_ACCESS_SECRET"),
		JWTRefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
		JWTAccessTTL:     jwtAccessTTL,
		JWTRefreshTTL:    jwtRefreshTTL,

		OTPTTL:              otpTTL,
		OTPMaxResendPerHour: viper.GetInt("OTP_MAX_RESEND_PER_HOUR"),

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
		{"DATABASE_URL", cfg.DatabaseURL},
		{"JWT_ACCESS_SECRET", cfg.JWTAccessSecret},
		{"JWT_REFRESH_SECRET", cfg.JWTRefreshSecret},
		{"REDIS_HOST", cfg.RedisHost},
		{"RABBITMQ_HOST", cfg.RabbitMQHost},
		{"RABBITMQ_ROUTING_KEY_EMAIL", cfg.RabbitMQRoutingKeyEmail},
		{"RABBITMQ_ROUTING_KEY_USER_SYNC", cfg.RabbitMQRoutingKeyUserSync},
	}

	for _, r := range required {
		if r.value == "" {
			return fmt.Errorf("config: %s is required", r.name)
		}
	}
	return nil
}
