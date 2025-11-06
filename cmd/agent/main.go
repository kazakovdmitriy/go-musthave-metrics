package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"go.uber.org/zap"
)

func main() {
	cfg := config.ParseAgentConfig()

	logg, err := logger.Initialize(cfg.LogLevel)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer func(logg *zap.Logger) {
		err := logg.Sync()
		if err != nil {

		}
	}(logg)

	app := agent.NewAppWithConfig(cfg, logg)

	app.Run()
	defer app.Stop()

	waitForShutdown(logg)
}

func waitForShutdown(logger *zap.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	logger.Info("received interrupt signal, shutting down...")
}
