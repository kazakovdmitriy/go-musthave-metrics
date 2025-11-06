package collector

import (
	"context"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"
	"sync"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

// metricsCollector отвечает за сбор метрик
type metricsCollector struct {
	metrics   model.MemoryMetrics
	mutex     sync.RWMutex
	pollCount int
	ticker    *time.Ticker
	logger    *zap.Logger
	ctx       context.Context
	providers []interfaces.MetricsProvider
}

// NewMetricsCollector создает новый сборщик метрик
func NewMetricsCollector(ctx context.Context, pollingInterval time.Duration, logger *zap.Logger, providers []interfaces.MetricsProvider) interfaces.MetricsCollector {
	return &metricsCollector{
		ticker:    time.NewTicker(pollingInterval),
		logger:    logger,
		ctx:       ctx,
		providers: providers,
	}
}

// Start запускает сбор метрик
func (mc *metricsCollector) Start() {
	go func() {
		for {
			select {
			case <-mc.ticker.C:
				mc.collectFromAllProviders()
			case <-mc.ctx.Done():
				return
			}
		}
	}()
}

// collectFromAllProviders собирает метрики из всех провайдеров
func (mc *metricsCollector) collectFromAllProviders() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	for i, provider := range mc.providers {
		newMetrics, err := provider.Collect(mc.ctx)
		if err != nil {
			mc.logger.Error("failed to collect metrics from provider",
				zap.Int("provider_index", i),
				zap.Error(err))
			continue
		}

		mc.mergeMetrics(newMetrics)
	}

	mc.pollCount++
}

// mergeMetrics объединяет метрики от разных провайдеров
func (mc *metricsCollector) mergeMetrics(newMetrics model.MemoryMetrics) {
	if mc.metrics.IsZero() {
		mc.metrics = newMetrics
		return
	}

	// Объединяем метрики, сохраняя не-zero значения
	if newMetrics.Alloc != 0 {
		mc.metrics.Alloc = newMetrics.Alloc
	}
	if newMetrics.BuckHashSys != 0 {
		mc.metrics.BuckHashSys = newMetrics.BuckHashSys
	}
	if newMetrics.Frees != 0 {
		mc.metrics.Frees = newMetrics.Frees
	}
	if newMetrics.GCCPUFraction != 0 {
		mc.metrics.GCCPUFraction = newMetrics.GCCPUFraction
	}
	if newMetrics.GCSys != 0 {
		mc.metrics.GCSys = newMetrics.GCSys
	}
	if newMetrics.HeapAlloc != 0 {
		mc.metrics.HeapAlloc = newMetrics.HeapAlloc
	}
	if newMetrics.HeapIdle != 0 {
		mc.metrics.HeapIdle = newMetrics.HeapIdle
	}
	if newMetrics.HeapInuse != 0 {
		mc.metrics.HeapInuse = newMetrics.HeapInuse
	}
	if newMetrics.HeapObjects != 0 {
		mc.metrics.HeapObjects = newMetrics.HeapObjects
	}
	if newMetrics.HeapReleased != 0 {
		mc.metrics.HeapReleased = newMetrics.HeapReleased
	}
	if newMetrics.HeapSys != 0 {
		mc.metrics.HeapSys = newMetrics.HeapSys
	}
	if newMetrics.LastGC != 0 {
		mc.metrics.LastGC = newMetrics.LastGC
	}
	if newMetrics.Lookups != 0 {
		mc.metrics.Lookups = newMetrics.Lookups
	}
	if newMetrics.MCacheInuse != 0 {
		mc.metrics.MCacheInuse = newMetrics.MCacheInuse
	}
	if newMetrics.MCacheSys != 0 {
		mc.metrics.MCacheSys = newMetrics.MCacheSys
	}
	if newMetrics.MSpanInuse != 0 {
		mc.metrics.MSpanInuse = newMetrics.MSpanInuse
	}
	if newMetrics.MSpanSys != 0 {
		mc.metrics.MSpanSys = newMetrics.MSpanSys
	}
	if newMetrics.Mallocs != 0 {
		mc.metrics.Mallocs = newMetrics.Mallocs
	}
	if newMetrics.NextGC != 0 {
		mc.metrics.NextGC = newMetrics.NextGC
	}
	if newMetrics.NumForcedGC != 0 {
		mc.metrics.NumForcedGC = newMetrics.NumForcedGC
	}
	if newMetrics.NumGC != 0 {
		mc.metrics.NumGC = newMetrics.NumGC
	}
	if newMetrics.OtherSys != 0 {
		mc.metrics.OtherSys = newMetrics.OtherSys
	}
	if newMetrics.PauseTotalNs != 0 {
		mc.metrics.PauseTotalNs = newMetrics.PauseTotalNs
	}
	if newMetrics.StackInuse != 0 {
		mc.metrics.StackInuse = newMetrics.StackInuse
	}
	if newMetrics.StackSys != 0 {
		mc.metrics.StackSys = newMetrics.StackSys
	}
	if newMetrics.Sys != 0 {
		mc.metrics.Sys = newMetrics.Sys
	}
	if newMetrics.TotalAlloc != 0 {
		mc.metrics.TotalAlloc = newMetrics.TotalAlloc
	}
	if newMetrics.RandomValue != 0 {
		mc.metrics.RandomValue = newMetrics.RandomValue
	}

	// Gopsutil метрики
	if newMetrics.TotalMemory != 0 {
		mc.metrics.TotalMemory = newMetrics.TotalMemory
	}
	if newMetrics.FreeMemory != 0 {
		mc.metrics.FreeMemory = newMetrics.FreeMemory
	}
	if len(newMetrics.CPUutilization) > 0 {
		mc.metrics.CPUutilization = newMetrics.CPUutilization
	}
}

// GetMetrics возвращает текущие метрики и счетчик
func (mc *metricsCollector) GetMetrics() (model.MemoryMetrics, int) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	return mc.metrics.Clone(), mc.pollCount
}

// ResetCount сбрасывает счетчик
func (mc *metricsCollector) ResetCount() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.pollCount = 0
}

// Stop останавливает сбор метрик
func (mc *metricsCollector) Stop() {
	if mc.ticker != nil {
		mc.ticker.Stop()
	}
}
