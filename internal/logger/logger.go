package logger

import (
	"fmt"

	"go.uber.org/zap"
)

// var Log *zap.Logger = zap.NewNop()

func Initialize(level string) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, fmt.Errorf("failed to determine logging level %w", err)
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger %w", err)
	}
	return logger, nil
}
