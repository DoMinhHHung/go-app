package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv  string
	AppPort string
	AppName string

	RabbitMQHost       string
	RabbitMQPort       int
	RabbitMQUsername   string
	RabbitMQPassword   string
	RabbitMQVHost      string
	RabbitMQExchange   string
	RabbitMQQueue      string
	RabbitMQRoutingKey string

	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
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

		RabbitMQHost:       viper.GetString("RABBITMQ_HOST"),
		RabbitMQPort:       viper.GetInt("RABBITMQ_PORT"),
		RabbitMQUsername:   viper.GetString("RABBITMQ_USERNAME"),
		RabbitMQPassword:   viper.GetString("RABBITMQ_PASSWORD"),
		RabbitMQVHost:      viper.GetString("RABBITMQ_VHOST"),
		RabbitMQExchange:   viper.GetString("RABBITMQ_EXCHANGE"),
		RabbitMQQueue:      viper.GetString("RABBITMQ_QUEUE"),
		RabbitMQRoutingKey: viper.GetString("RABBITMQ_ROUTING_KEY"),

		SMTPHost:     viper.GetString("SMTP_HOST"),
		SMTPPort:     viper.GetInt("SMTP_PORT"),
		SMTPUsername: viper.GetString("SMTP_USERNAME"),
		SMTPPassword: viper.GetString("SMTP_PASSWORD"),
		SMTPFrom:     viper.GetString("SMTP_FROM"),
	}

	return cfg, validate(cfg)
}

func validate(cfg *Config) error {
	required := []struct {
		name  string
		value string
	}{
		{"RABBITMQ_HOST", cfg.RabbitMQHost},
		{"RABBITMQ_EXCHANGE", cfg.RabbitMQExchange},
		{"RABBITMQ_QUEUE", cfg.RabbitMQQueue},
		{"RABBITMQ_ROUTING_KEY", cfg.RabbitMQRoutingKey},
		{"SMTP_HOST", cfg.SMTPHost},
		{"SMTP_USERNAME", cfg.SMTPUsername},
		{"SMTP_FROM", cfg.SMTPFrom},
	}

	for _, r := range required {
		if r.value == "" {
			return fmt.Errorf("config: %s is required", r.name)
		}
	}
	return nil
}
