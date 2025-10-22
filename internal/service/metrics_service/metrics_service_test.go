package metrics_service

import (
	"context"
	"testing"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestMetricsService_UpdateGauge(t *testing.T) {
	storage := mocks.NewMockStorage()
	service := NewMetricService(storage)

	ctx := context.Background()
	err := service.UpdateGauge(ctx, "test_gauge", 123.45)
	assert.NoError(t, err)

	value, ok := storage.GetGauge(ctx, "test_gauge")
	assert.True(t, ok)
	assert.Equal(t, 123.45, value)
}

func TestMetricsService_UpdateCounter(t *testing.T) {
	storage := mocks.NewMockStorage()
	service := NewMetricService(storage)

	ctx := context.Background()
	err := service.UpdateCounter(ctx, "test_counter", 10)
	assert.NoError(t, err)

	value, ok := storage.GetCounter(ctx, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, int64(10), value)
}

func TestMetricsService_GetGauge(t *testing.T) {
	storage := mocks.NewMockStorage()
	ctx := context.Background()
	storage.UpdateGauge(ctx, "existing_gauge", 99.99)
	service := NewMetricService(storage)

	value, err := service.GetGauge(ctx, "existing_gauge")
	assert.NoError(t, err)
	assert.Equal(t, 99.99, value)

	_, err = service.GetGauge(ctx, "missing_gauge")
	assert.Error(t, err)
	assert.Equal(t, "gauge metric not found", err.Error())
}

func TestMetricsService_GetCounter(t *testing.T) {
	storage := mocks.NewMockStorage()
	ctx := context.Background()
	storage.UpdateCounter(ctx, "existing_counter", 5)
	service := NewMetricService(storage)

	value, err := service.GetCounter(ctx, "existing_counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(5), value)

	_, err = service.GetCounter(ctx, "missing_counter")
	assert.Error(t, err)
	assert.Equal(t, "counter metric not found", err.Error())
}
