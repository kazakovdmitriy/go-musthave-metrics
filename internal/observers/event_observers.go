package observers

import (
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
)

// EventObserver - интерфейс для конкретных наблюдателей
type EventObserver interface {
	OnMetricProcessed(event model.MetricProcessedEvent)
}
