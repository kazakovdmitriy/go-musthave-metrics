package main

import (
	"context"
	"fmt"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/utils/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/utils/logger"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent"
)

// Глобальные переменные для информации о сборке
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	printBuildInfo()

	cfg, err := config.ParseAgentConfig()
	if err != nil {
		log.Fatal(err)
	}

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

func printBuildInfo() {
	fmt.Fprintf(os.Stdout, "Build version: %s\n", buildVersion)
	fmt.Fprintf(os.Stdout, "Build date: %s\n", buildDate)
	fmt.Fprintf(os.Stdout, "Build commit: %s\n", buildCommit)
}
