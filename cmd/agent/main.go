package main

import (
	"context"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"log"
	"os/signal"
	"syscall"
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

	app := agent.NewAppWithConfig(ctx, cfg, logg)

	app.Run(ctx)
	app.Wait()

	logg.Info("application shutdown complete")
}
