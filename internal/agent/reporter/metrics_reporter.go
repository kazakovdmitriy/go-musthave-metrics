package reporter

import (
	"context"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"
	"sync"
	"time"

	"go.uber.org/zap"
)

// metricsReporter отвечает за отправку метрик
type metricsReporter struct {
	sender    interfaces.MetricsSender
	ticker    *time.Ticker
	logger    *zap.Logger
	ctx       context.Context
	collector interfaces.MetricsCollector
	wg        sync.WaitGroup
}

// NewMetricsReporter создает новый репортер метрик
func NewMetricsReporter(
	ctx context.Context,
	collector interfaces.MetricsCollector,
	sender interfaces.MetricsSender,
	reportInterval time.Duration,
	logger *zap.Logger,
) interfaces.MetricsReporter {
	return &metricsReporter{
		sender:    sender,
		ticker:    time.NewTicker(reportInterval),
		logger:    logger,
		ctx:       ctx,
		collector: collector,
	}
}

// Start запускает отправку метрик
func (mr *metricsReporter) Start() {
	mr.wg.Add(1)

	go func() {
		defer mr.wg.Done()
		for {
			select {
			case <-mr.ticker.C:
				mr.report()
			case <-mr.ctx.Done():
				return
			}
		}
	}()
}

// report выполняет отправку метрик
func (mr *metricsReporter) report() {
	metrics, count := mr.collector.GetMetrics()

	err := mr.sender.Send(mr.ctx, metrics, int64(count), mr.logger)
	if err != nil {
		mr.logger.Error("error sending metrics", zap.Error(err))
	} else {
		mr.collector.ResetCount()
		mr.logger.Info("metrics submitted to worker pool successfully")
	}
}

// Stop останавливает отправку метрик
func (mr *metricsReporter) Stop() {
	if mr.ticker != nil {
		mr.ticker.Stop()
	}
	if mr.sender != nil {
		mr.sender.Stop()
	}
	mr.wg.Wait()
}
