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
	}

	for _, r := range required {
		if r.value == "" {
			return fmt.Errorf("config: %s is required", r.name)
		}
	}
	return nil
}
