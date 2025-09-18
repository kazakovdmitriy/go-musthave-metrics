package service

import (
	"errors"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/repository"
)

type metricsService struct {
	storage repository.Storage
}

func NewMetricService(storage repository.Storage) MetricsService {
	return &metricsService{
		storage: storage,
	}
}

func (s *metricsService) UpdateGauge(name string, value float64) error {
	s.storage.UpdateGauge(name, value)
	return nil
}

func (s *metricsService) UpdateCounter(name string, value int64) error {
	s.storage.UpdateCounter(name, value)
	return nil
}

func (s *metricsService) GetGauge(name string) (float64, error) {
	value, exist := s.storage.GetGauge(name)
	if !exist {
		return 0, errors.New("gauge metric not found")
	}

	return value, nil
}

func (s *metricsService) GetCounter(name string) (int64, error) {
	value, exist := s.storage.GetCounter(name)
	if !exist {
		return 0, errors.New("counter metric not found")
	}

	return value, nil
}
