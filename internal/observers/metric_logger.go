package observers

import (
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

type MetricLogger struct {
	logger *zap.Logger
}

func NewMetricLogger(logger *zap.Logger) *MetricLogger {
	return &MetricLogger{
		logger: logger,
	}
}

func (m *MetricLogger) OnMetricProcessed(event model.MetricProcessedEvent) {
	m.logger.Info("Metric processed", zap.Any("event", event))
}
