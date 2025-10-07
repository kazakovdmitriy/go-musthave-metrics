package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"go.uber.org/zap"
)

func main() {
	cfg := config.ParseAgentConfig()

	if err := logger.Initialise(cfg.LogLevel); err != nil {
		panic(err)
	}

	client := agent.NewClient(cfg.ServerAddr)

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
				metricsMutex.RUnlock()

				_, err := agent.SendMetrics(client, currentMetrics)
				if err != nil {
					logger.Log.Error("error from server", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	<-sigChan
	logger.Log.Info("received interrupt signal. shutting down gracefully...")

	cancel()
	pollTicker.Stop()
	reportTicker.Stop()

	wg.Wait()
	logger.Log.Info("agent shutdown completed")
}
