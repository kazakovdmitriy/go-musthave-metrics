package service

import (
	"context"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
)

type Storage interface {
	UpdateGauge(ctx context.Context, name string, value float64)
	UpdateCounter(ctx context.Context, name string, value int64)
	UpdateMetrics(ctx context.Context, metrics []model.Metrics) error
	GetGauge(ctx context.Context, name string) (float64, bool)
	GetCounter(ctx context.Context, name string) (int64, bool)
	GetAllMetrics(ctx context.Context) (string, error)
	Close() error
}
