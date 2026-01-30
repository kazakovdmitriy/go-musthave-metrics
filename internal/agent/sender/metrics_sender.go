package sender

import (
	"context"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

// metricsSender отправляет метрики на сервер используя worker pool
type metricsSender struct {
	workerPool     *WorkerPool
	metricsService *metricsService
	log            *zap.Logger
}

// NewMetricsSender создает новый отправитель метрик
func NewMetricsSender(client interfaces.HTTPClient, workers int, queueSize int, logger *zap.Logger) interfaces.MetricsSender {
	if workers <= 0 {
		logger.Info("creating unlimited sender (no worker pool)")
		return newUnlimitedSender(client, logger)
	}

	metricsService := newMetricsService(client, logger)

	workerPool := NewWorkerPool(workers, queueSize, logger)
	workerPool.Start()

	return &metricsSender{
		workerPool:     workerPool,
		metricsService: metricsService,
		log:            logger,
	}
}

// Send отправляет метрики на сервер через worker pool
func (ms *metricsSender) Send(ctx context.Context, metrics model.MemoryMetrics, deltaCounter int64) error {
	task := func() error {
		return ms.metricsService.send(ctx, metrics, deltaCounter)
	}

	submitted := ms.workerPool.Submit(task)
	if !submitted {
		ms.log.Warn("failed to submit metricshandler task to worker pool")
	}
	return nil
}

// Stop останавливает отправитель метрик
func (ms *metricsSender) Stop() {
	if ms.workerPool != nil {
		ms.workerPool.Stop()
	}
}
