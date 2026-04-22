package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const requestIDKey contextKey = "request_id"

func New(appEnv string) (*zap.Logger, error) {
	level := zap.InfoLevel
	if appEnv != "production" {
		level = zap.DebugLevel
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      appEnv != "production",
		Encoding:         "json",
		EncoderConfig:    encoderCfg,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return cfg.Build()
}

func NewContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func FromContext(ctx context.Context, base *zap.Logger) *zap.Logger {
	reqID, ok := ctx.Value(requestIDKey).(string)
	if !ok || reqID == "" {
		return base
	}
	return base.With(zap.String("request_id", reqID))
}
