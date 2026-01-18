package interfaces

import (
	"context"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
)

// HTTPClient интерфейс для HTTP клиента
type HTTPClient interface {
	Post(ctx context.Context, endpoint string, body interface{}) ([]byte, error)
	Get(ctx context.Context, endpoint string) ([]byte, error)
}

// MetricsSender интерфейс для отправителя метрик
type MetricsSender interface {
	Send(ctx context.Context, metrics model.MemoryMetrics, deltaCounter int64) error
	Stop()
}

// MetricsCollector интерфейс для сборщика метрик
type MetricsCollector interface {
	Start()
	Stop()
	GetMetrics() (model.MemoryMetrics, int)
	ResetCount()
}

// MetricsReporter интерфейс для репортера метрик
type MetricsReporter interface {
	Start()
	Stop()
}

// MetricsProvider интерфейс для поставщика метрик
type MetricsProvider interface {
	Collect(ctx context.Context) (model.MemoryMetrics, error)
}
