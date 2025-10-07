package agent

import (
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

func SendMetrics(client *Client, metrics MemoryMetrics) ([]byte, error) {

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
	return nil, nil
}
