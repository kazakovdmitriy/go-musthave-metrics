package main

import (
	"context"
	"fmt"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/utils/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/utils/logger"
	"net/http"
	"os"
	"sync"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/server"
	"go.uber.org/zap"
)

// Глобальные переменные для информации о сборке
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	printBuildInfo()

	cfg := config.ParseServerConfig()

	log, err := logger.Initialize(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	// === ЗАПУСК PPROF НА ОТДЕЛЬНОМ ПОРТУ ===
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("pprof server starting on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Error("pprof server failed", zap.Error(err))
		}
	}()
	// ======================================

	server, err := server.NewApp(cfg, log)
	if err != nil {
		log.Fatal("failed to create application", zap.Error(err))
	}
	defer server.Close()

	ctx := context.Background()
	if err := server.Run(ctx); err != nil {
		log.Error("application failed", zap.Error(err))
		os.Exit(1)
	}

	wg.Wait()
}

func printBuildInfo() {
	fmt.Fprintf(os.Stdout, "Build version: %s\n", buildVersion)
	fmt.Fprintf(os.Stdout, "Build date: %s\n", buildDate)
	fmt.Fprintf(os.Stdout, "Build commit: %s\n", buildCommit)
}
