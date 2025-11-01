package agent

import (
	"context"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

func SendMetrics(
	ctx context.Context,
	client *Client,
	metrics MemoryMetrics,
	deltaCounter int64,
	log *zap.Logger,
) ([]byte, error) {

	metricsMap := metrics.ToMap()

	if len(metricsMap) == 0 && deltaCounter == 0 {
		log.Info("no metrics to send")
		return nil, nil
	}

	var batch []model.Metrics

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
		log.Info("no metrics to send after filtering")
		return nil, nil
	}

	response, err := client.Post(ctx, "/updates/", batch)
	if err != nil {
		log.Error(
			"failed to send metrics batch",
			zap.Int("metrics_count", len(batch)),
			zap.Error(err),
		)
		return nil, err
	}

	log.Debug(
		"successfully sent metrics batch",
		zap.Int("metrics_count", len(batch)),
	)

	return response, nil
}
