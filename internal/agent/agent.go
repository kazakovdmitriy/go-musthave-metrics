package agent

import (
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

func SendMetrics(client *Client, metrics MemoryMetrics, deltaCounter int64) ([]byte, error) {

	metricsMap := metrics.ToMap()

	for name, value := range metricsMap {
		body := model.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &value,
		}
		_, err := client.Post("/update", body)
		if err != nil {
			logger.Log.Error(
				"failed to send metric",
				zap.String("metric", name),
				zap.Float64("value", value),
			)
			return nil, err
		}
	}

	counterBody := model.Metrics{
		ID:    "PollCount",
		MType: "counter",
		Delta: &deltaCounter,
	}
	_, err := client.Post("/update", counterBody)
	if err != nil {
		logger.Log.Error(
			"failed to send counter",
			zap.String("metric", counterBody.ID),
			zap.Int64("value", deltaCounter),
		)
		return nil, err
	}
	return nil, nil
}
