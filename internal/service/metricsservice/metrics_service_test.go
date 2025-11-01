package metricsservice

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestMetricsService_UpdateGauge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mocks.NewMockStorage(ctrl)
	service := NewMetricService(storage)

	ctx := context.Background()
	storage.EXPECT().UpdateGauge(ctx, "test_gauge", 123.45).Times(1)

	err := service.UpdateGauge(ctx, "test_gauge", 123.45)
	assert.NoError(t, err)
}

func TestMetricsService_UpdateCounter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mocks.NewMockStorage(ctrl)
	service := NewMetricService(storage)

	ctx := context.Background()
	storage.EXPECT().UpdateCounter(ctx, "test_counter", int64(10)).Times(1)

	err := service.UpdateCounter(ctx, "test_counter", 10)
	assert.NoError(t, err)
}

func TestMetricsService_GetGauge_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mocks.NewMockStorage(ctrl)
	service := NewMetricService(storage)

	ctx := context.Background()
	storage.EXPECT().GetGauge(ctx, "existing_gauge").Return(99.99, true)

	value, err := service.GetGauge(ctx, "existing_gauge")
	assert.NoError(t, err)
	assert.Equal(t, 99.99, value)
}

func TestMetricsService_GetGauge_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mocks.NewMockStorage(ctrl)
	service := NewMetricService(storage)

	ctx := context.Background()
	storage.EXPECT().GetGauge(ctx, "missing_gauge").Return(0.0, false)

	_, err := service.GetGauge(ctx, "missing_gauge")
	assert.Error(t, err)
	assert.Equal(t, "gauge metric not found", err.Error())
}

func TestMetricsService_GetCounter_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mocks.NewMockStorage(ctrl)
	service := NewMetricService(storage)

	ctx := context.Background()
	storage.EXPECT().GetCounter(ctx, "existing_counter").Return(int64(5), true)

	value, err := service.GetCounter(ctx, "existing_counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(5), value)
}

func TestMetricsService_GetCounter_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mocks.NewMockStorage(ctrl)
	service := NewMetricService(storage)

	ctx := context.Background()
	storage.EXPECT().GetCounter(ctx, "missing_counter").Return(int64(0), false)

	_, err := service.GetCounter(ctx, "missing_counter")
	assert.Error(t, err)
	assert.Equal(t, "counter metric not found", err.Error())
}
