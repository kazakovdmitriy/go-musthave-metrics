package sender

import (
	"context"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

// metricsSender отправляет метрики на сервер используя worker pool
type metricsSender struct {
	workerPool *WorkerPool
}

// NewMetricsSender создает новый отправитель метрик
func NewMetricsSender(client interfaces.HTTPClient, workers int, queueSize int, logger *zap.Logger) interfaces.MetricsSender {
	// Если workers = 0, используем неограниченный режим
	if workers <= 0 {
		logger.Info("creating unlimited sender (no worker pool)")
		return NewUnlimitedSender(client, logger)
	}

	workerPool := NewWorkerPool(client, workers, queueSize, logger)
	workerPool.Start()

	return &metricsSender{
		workerPool: workerPool,
	}
}

// Send отправляет метрики на сервер через worker pool
func (ms *metricsSender) Send(ctx context.Context, metrics model.MemoryMetrics, deltaCounter int64, logger *zap.Logger) error {
	task := SendTask{
		Metrics: metrics,
		Count:   deltaCounter,
	}

	ms.workerPool.Submit(task)
	return nil
}

// Stop останавливает отправитель метрик
func (ms *metricsSender) Stop() {
	if ms.workerPool != nil {
		ms.workerPool.Stop()
	}
}
