package service

import (
	"testing"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestMetricsService_UpdateGauge(t *testing.T) {
	storage := repository.NewMockStorage()
	service := NewMetricService(storage)

	err := service.UpdateGauge("test_gauge", 123.45)
	assert.NoError(t, err)

	value, ok := storage.GetGauge("test_gauge")
	assert.True(t, ok)
	assert.Equal(t, 123.45, value)
}

func TestMetricsService_UpdateCounter(t *testing.T) {
	storage := repository.NewMockStorage()
	service := NewMetricService(storage)

	err := service.UpdateCounter("test_counter", 10)
	assert.NoError(t, err)

	value, ok := storage.GetCounter("test_counter")
	assert.True(t, ok)
	assert.Equal(t, int64(10), value)
}

func TestMetricsService_GetGauge(t *testing.T) {
	storage := repository.NewMockStorage()
	storage.UpdateGauge("existing_gauge", 99.99)
	service := NewMetricService(storage)

	value, err := service.GetGauge("existing_gauge")
	assert.NoError(t, err)
	assert.Equal(t, 99.99, value)

	_, err = service.GetGauge("missing_gauge")
	assert.Error(t, err)
	assert.Equal(t, "gauge metric not found", err.Error())
}

func TestMetricsService_GetCounter(t *testing.T) {
	storage := repository.NewMockStorage()
	storage.UpdateCounter("existing_counter", 5)
	service := NewMetricService(storage)

	value, err := service.GetCounter("existing_counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(5), value)

	_, err = service.GetCounter("missing_counter")
	assert.Error(t, err)
	assert.Equal(t, "counter metric not found", err.Error())
}
