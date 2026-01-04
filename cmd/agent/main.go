package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
)

func main() {
	cfg := config.ParseAgentConfig()

	logg, err := logger.Initialize(cfg.LogLevel)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer logg.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := agent.NewAppWithConfig(ctx, cfg, logg)
	if err != nil {
		log.Fatal(err.Error())
	}

	app.Run(ctx)
	app.Wait()

	logg.Info("application shutdown complete")
}
