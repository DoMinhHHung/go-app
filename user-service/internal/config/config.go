package config

import (
	"fmt"

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

	RabbitMQHost       string
	RabbitMQPort       int
	RabbitMQUsername   string
	RabbitMQPassword   string
	RabbitMQVHost      string
	RabbitMQExchange   string
	RabbitMQQueue      string
	RabbitMQRoutingKey string

	JWTAccessSecret string
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config: read .env: %w", err)
	}

	cfg := &Config{
		AppEnv:  viper.GetString("APP_ENV"),
		AppPort: viper.GetString("APP_PORT"),
		AppName: viper.GetString("APP_NAME"),

		DatabaseURL: viper.GetString("DATABASE_URL"),

		RedisHost:     viper.GetString("REDIS_HOST"),
		RedisPort:     viper.GetString("REDIS_PORT"),
		RedisPassword: viper.GetString("REDIS_PASSWORD"),

		RabbitMQHost:       viper.GetString("RABBITMQ_HOST"),
		RabbitMQPort:       viper.GetInt("RABBITMQ_PORT"),
		RabbitMQUsername:   viper.GetString("RABBITMQ_USERNAME"),
		RabbitMQPassword:   viper.GetString("RABBITMQ_PASSWORD"),
		RabbitMQVHost:      viper.GetString("RABBITMQ_VHOST"),
		RabbitMQExchange:   viper.GetString("RABBITMQ_EXCHANGE"),
		RabbitMQQueue:      viper.GetString("RABBITMQ_QUEUE"),
		RabbitMQRoutingKey: viper.GetString("RABBITMQ_ROUTING_KEY"),

		JWTAccessSecret: viper.GetString("JWT_ACCESS_SECRET"),
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
		{"RABBITMQ_HOST", cfg.RabbitMQHost},
		{"RABBITMQ_QUEUE", cfg.RabbitMQQueue},
		{"RABBITMQ_ROUTING_KEY", cfg.RabbitMQRoutingKey},
	}

	for _, r := range required {
		if r.value == "" {
			return fmt.Errorf("config: %s is required", r.name)
		}
	}
	return nil
}
