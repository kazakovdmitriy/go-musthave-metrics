package repository

import (
	"sync"
	"testing"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMemStorage_UpdateAndGetGauge(t *testing.T) {
	storage := NewMemStorage(&config.ServerFlags{})
	name := "test_gauge"
	value := 42.5

	storage.UpdateGauge(name, value)
	got, ok := storage.GetGauge(name)

	assert.True(t, ok)
	assert.Equal(t, value, got)
}

func TestMemStorage_UpdateAndGetCounter(t *testing.T) {
	storage := NewMemStorage(&config.ServerFlags{})
	name := "test_counter"
	value := int64(100)

	storage.UpdateCounter(name, value)
	got, ok := storage.GetCounter(name)

	assert.True(t, ok)
	assert.Equal(t, value, got)
}

func TestMemStorage_UpdateCounterIncrement(t *testing.T) {
	storage := NewMemStorage(&config.ServerFlags{})
	name := "test_counter"

	storage.UpdateCounter(name, 10)
	storage.UpdateCounter(name, 5)

	got, ok := storage.GetCounter(name)
	assert.True(t, ok)
	assert.Equal(t, int64(15), got)
}

func TestMemStorage_GetNonExistent(t *testing.T) {
	storage := NewMemStorage(&config.ServerFlags{})

	_, ok := storage.GetGauge("nonexistent")
	assert.False(t, ok)

	_, ok = storage.GetCounter("nonexistent")
	assert.False(t, ok)
}

func TestMemStorage_GetAllMetrics(t *testing.T) {
	storage := NewMemStorage(&config.ServerFlags{})
	storage.UpdateGauge("gauge1", 3.14)
	storage.UpdateCounter("counter1", 42)

	result, err := storage.GetAllMetrics()
	assert.NoError(t, err)
	assert.Contains(t, result, "<li>gauge1 = 3.140000</li>")
	assert.Contains(t, result, "<li>counter1 = 42</li>")
}

func TestMemStorage_GetAllMetrics_Empty(t *testing.T) {
	storage := NewMemStorage(&config.ServerFlags{})
	_, err := storage.GetAllMetrics()
	assert.Error(t, err)
	assert.Equal(t, "no metrics found", err.Error())
}

func TestMemStorage_ConcurrentAccess(t *testing.T) {
	storage := NewMemStorage(&config.ServerFlags{})
	var wg sync.WaitGroup
	const workers = 10
	const increments = 100

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				storage.UpdateCounter("counter", 1)
				storage.UpdateGauge("gauge", float64(j))
			}
		}()
	}
	wg.Wait()

	counter, ok := storage.GetCounter("counter")
	assert.True(t, ok)
	assert.Equal(t, int64(workers*increments), counter)
}
