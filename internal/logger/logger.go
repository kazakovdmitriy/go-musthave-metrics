package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"

	"go.uber.org/zap"
)

func Initialize(level string) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, fmt.Errorf("failed to determine logging level %w", err)
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	cfg.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger %w", err)
	}

	return logger.With(zap.Float64("timestamp", float64(time.Now().UnixNano())/1e9)), nil
}
