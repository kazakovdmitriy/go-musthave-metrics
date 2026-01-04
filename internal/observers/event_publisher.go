package observers

import (
	"context"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
)

type EventPublisher interface {
	Publish(ctx context.Context, event model.MetricProcessedEvent) error
	Register(observer EventObserver)
	Unregister(observer EventObserver)
}
