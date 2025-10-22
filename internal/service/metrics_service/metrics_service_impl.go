package metrics_service

import (
	"context"
	"errors"

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
	s.storage.UpdateGauge(ctx, name, value)
	return nil
}

func (s *metricsService) UpdateCounter(ctx context.Context, name string, value int64) error {
	s.storage.UpdateCounter(ctx, name, value)
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
