package metricsservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/observers"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
)

type metricsService struct {
	storage  service.Storage
	eventPub observers.EventPublisher
}

func NewMetricService(
	storage service.Storage,
	eventPub observers.EventPublisher,
) *metricsService {
	return &metricsService{
		storage:  storage,
		eventPub: eventPub,
	}
}

func (s *metricsService) UpdateGauge(ctx context.Context, name string, value float64) error {
	return s.storage.UpdateGauge(ctx, name, value)
}

func (s *metricsService) UpdateCounter(ctx context.Context, name string, value int64) error {
	return s.storage.UpdateCounter(ctx, name, value)
}

func (s *metricsService) UpdateMetrics(ctx context.Context, metrics []model.Metrics, ipAddr string) error {
	if err := s.storage.UpdateMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("failed to save metrics in storage: %w", err)
	}

	var metricsArr []string
	for _, m := range metrics {
		metricsArr = append(metricsArr, m.MType)
	}

	now := time.Now()
	event := model.MetricProcessedEvent{
		Timestamp: now,
		Ts:        now.UnixMilli(),
		Metrics:   metricsArr,
		IpAddr:    ipAddr,
	}

	go func() {
		err := s.eventPub.Publish(ctx, event)
		if err != nil {
			fmt.Printf("failed to publish metrics: %v\n", err)
		}
	}()

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
