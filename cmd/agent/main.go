package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares/signer"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service/signerservice"
	"go.uber.org/zap"
)

func main() {
	cfg := config.ParseAgentConfig()

	logg, err := logger.Initialize(cfg.LogLevel)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer logg.Sync()

	var signerService signer.Signer = nil
	if cfg.SecretKet != "" {
		signerService = signerservice.NewSHA256Signer(cfg.SecretKet)
	}
	client := agent.NewClient(cfg.ServerAddr, signerService, logg)

	polingInterval := time.Duration(cfg.PollingInterval) * time.Second
	reportInterval := time.Duration(cfg.ReportInterval) * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var metricsMutex sync.RWMutex
	metrics, err := agent.GetMetrics(ctx)
	if err != nil {
		return
	}

	pollTicker := time.NewTicker(polingInterval)
	defer pollTicker.Stop()

	poolCount := 0

	go func() {
		for {
			select {
			case <-pollTicker.C:
				newMetrics, err := agent.GetMetrics(ctx)
				if err != nil {
					return
				}
				metricsMutex.Lock()
				metrics = newMetrics
				poolCount += 1
				metricsMutex.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	reportTicker := time.NewTicker(reportInterval)
	defer reportTicker.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-reportTicker.C:
				metricsMutex.RLock()
				currentMetrics := metrics
				deltaCount := poolCount
				metricsMutex.RUnlock()

				_, err := agent.SendMetrics(ctx, client, currentMetrics, int64(deltaCount), logg)
				if err != nil {
					logg.Error("error from server", zap.Error(err))
				} else {
					poolCount = 0
				}
				logg.Info("metrics sent successfully")
			case <-ctx.Done():
				return
			}
		}
	}()

	<-sigChan
	logg.Info("received interrupt signal. shutting down gracefully...")

	cancel()
	pollTicker.Stop()
	reportTicker.Stop()

	wg.Wait()
	logg.Info("agent shutdown completed")
}
