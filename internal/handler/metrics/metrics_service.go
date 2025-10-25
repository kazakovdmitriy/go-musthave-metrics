package metrics

import (
	"context"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
)

type MetricsService interface {
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, value int64) error
	UpdateMetrics(ctx context.Context, metrics []model.Metrics) error
	GetGauge(ctx context.Context, name string) (float64, error)
	GetCounter(ctx context.Context, name string) (int64, error)
}
