package metricsservice

import (
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/observers"
)

type EventPublisher interface {
	Publish(event model.MetricProcessedEvent) error
	Register(observer observers.EventObserver)
}
