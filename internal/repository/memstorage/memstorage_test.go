package memstorage

import (
	"context"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/utils/config"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestMemStorage_UpdateAndGetGauge(t *testing.T) {
	logger := zaptest.NewLogger(t)
	storage := NewMemStorage(&config.ServerFlags{}, logger)
	name := "test_gauge"
	value := 42.5

	ctx := context.Background()
	storage.UpdateGauge(ctx, name, value)
	got, ok := storage.GetGauge(ctx, name)

	assert.True(t, ok)
	assert.Equal(t, value, got)
}

func TestMemStorage_UpdateAndGetCounter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	storage := NewMemStorage(&config.ServerFlags{}, logger)
	name := "test_counter"
	value := int64(100)

	ctx := context.Background()
	storage.UpdateCounter(ctx, name, value)
	got, ok := storage.GetCounter(ctx, name)

	assert.True(t, ok)
	assert.Equal(t, value, got)
}

func TestMemStorage_UpdateCounterIncrement(t *testing.T) {
	logger := zaptest.NewLogger(t)
	storage := NewMemStorage(&config.ServerFlags{}, logger)
	name := "test_counter"

	ctx := context.Background()
	storage.UpdateCounter(ctx, name, 10)
	storage.UpdateCounter(ctx, name, 5)

	got, ok := storage.GetCounter(ctx, name)
	assert.True(t, ok)
	assert.Equal(t, int64(15), got)
}

func TestMemStorage_GetNonExistent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	storage := NewMemStorage(&config.ServerFlags{}, logger)

	ctx := context.Background()
	_, ok := storage.GetGauge(ctx, "nonexistent")
	assert.False(t, ok)

	_, ok = storage.GetCounter(ctx, "nonexistent")
	assert.False(t, ok)
}

func TestMemStorage_GetAllMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	storage := NewMemStorage(&config.ServerFlags{}, logger)

	ctx := context.Background()
	storage.UpdateGauge(ctx, "gauge1", 3.14)
	storage.UpdateCounter(ctx, "counter1", 42)

	result, err := storage.GetAllMetrics(ctx)
	assert.NoError(t, err)
	assert.Contains(t, result, "<li>gauge1 = 3.140000</li>")
	assert.Contains(t, result, "<li>counter1 = 42</li>")
}

func TestMemStorage_GetAllMetrics_Empty(t *testing.T) {
	logger := zaptest.NewLogger(t)
	storage := NewMemStorage(&config.ServerFlags{}, logger)

	ctx := context.Background()
	_, err := storage.GetAllMetrics(ctx)
	assert.Error(t, err)
	assert.Equal(t, "no metricshandler found", err.Error())
}

func TestMemStorage_ConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	storage := NewMemStorage(&config.ServerFlags{}, logger)
	var wg sync.WaitGroup
	const workers = 10
	const increments = 100

	ctx := context.Background()
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				storage.UpdateCounter(ctx, "counter", 1)
				storage.UpdateGauge(ctx, "gauge", float64(j))
			}
		}()
	}
	wg.Wait()

	counter, ok := storage.GetCounter(ctx, "counter")
	assert.True(t, ok)
	assert.Equal(t, int64(workers*increments), counter)
}
