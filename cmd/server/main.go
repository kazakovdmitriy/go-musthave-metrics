package main

import (
	"fmt"
	"os"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/server"
	"go.uber.org/zap"
)

func main() {
	cfg := config.ParseServerConfig()

	log, err := logger.Initialize(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	server, err := server.NewApp(cfg, log)
	if err != nil {
		log.Fatal("failed to create application", zap.Error(err))
	}
	defer server.Close()

	if err := server.Run(); err != nil {
		log.Error("application failed", zap.Error(err))
		os.Exit(1)
	}
}
