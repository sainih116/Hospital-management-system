// Package logger provides a centralized, structured logger for the
// application built on top of zap.
package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds a zap.Logger configured for the given environment and level.
// In production it emits JSON logs; in development it emits a readable
// console format.
func New(env, level string) (*zap.Logger, error) {
	var zapCfg zap.Config

	if strings.EqualFold(env, "production") {
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	zapCfg.Level = zap.NewAtomicLevelAt(parseLevel(level))

	log, err := zapCfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		return nil, err
	}

	return log, nil
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
