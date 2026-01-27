package sender

import (
	"context"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

// metricsService содержит общую логику отправки метрик
type metricsService struct {
	client    interfaces.HTTPClient
	logger    *zap.Logger
	batchPool *MetricsBatchPool
}

func newMetricsService(client interfaces.HTTPClient, logger *zap.Logger) *metricsService {
	return &metricsService{
		client:    client,
		logger:    logger,
		batchPool: NewMetricsBatchPool(20),
	}
}

// send - общая логика отправки, которую используют все отправители
func (ms *metricsService) send(ctx context.Context, metrics model.MemoryMetrics, deltaCounter int64) error {
	metricsMap := metrics.ToMap()

	if len(metricsMap) == 0 && deltaCounter == 0 {
		ms.logger.Info("no metricshandler to send")
		return nil
	}

	// Берем батч из пула
	batchWrapper := ms.batchPool.GetBatch()
	defer ms.batchPool.PutBatch(batchWrapper)

	batch := batchWrapper.Slice

	for name, value := range metricsMap {
		valueCopy := value
		batch = append(batch, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &valueCopy,
		})
	}

	if deltaCounter != 0 {
		deltaCopy := deltaCounter
		batch = append(batch, model.Metrics{
			ID:    "PollCount",
			MType: model.Counter,
			Delta: &deltaCopy,
		})
	}

	if len(batch) == 0 {
		ms.logger.Info("no metricshandler to send after filtering")
		return nil
	}

	_, err := ms.client.Post(ctx, "/updates/", batch)
	if err != nil {
		ms.logger.Error(
			"failed to send metricshandler batch",
			zap.Int("metrics_count", len(batch)),
			zap.Error(err),
		)
		return err
	}

	ms.logger.Debug(
		"successfully sent metricshandler batch",
		zap.Int("metrics_count", len(batch)),
	)

	return nil
}
