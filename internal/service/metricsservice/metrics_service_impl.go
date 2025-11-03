package metricsservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
)

type metricsService struct {
	storage service.Storage
}

func NewMetricService(storage service.Storage) *metricsService {
	return &metricsService{
		storage: storage,
	}
}

func (s *metricsService) UpdateGauge(ctx context.Context, name string, value float64) error {
	return s.storage.UpdateGauge(ctx, name, value)
}

func (s *metricsService) UpdateCounter(ctx context.Context, name string, value int64) error {
	return s.storage.UpdateCounter(ctx, name, value)
}

func (s *metricsService) UpdateMetrics(ctx context.Context, metrics []model.Metrics) error {
	if err := s.storage.UpdateMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("failed to save metrics in storage: %w", err)
	}
	return nil
}

func (s *metricsService) GetGauge(ctx context.Context, name string) (float64, error) {
	value, exist := s.storage.GetGauge(ctx, name)
	if !exist {
		return 0, errors.New("gauge metric not found")
	}

	return value, nil
}

func (s *metricsService) GetCounter(ctx context.Context, name string) (int64, error) {
	value, exist := s.storage.GetCounter(ctx, name)
	if !exist {
		return 0, errors.New("counter metric not found")
	}

	return value, nil
}
