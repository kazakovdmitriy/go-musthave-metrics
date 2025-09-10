package repository

import (
	models "github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
)

type memStorage struct {
	data map[string]models.Metrics
}

func NewMemStorage() Storage {
	return &memStorage{
		data: make(map[string]models.Metrics),
	}
}

func (m *memStorage) UpdateGauge(name string, value float64) {
	m.data[name] = models.Metrics{
		ID:    "id",
		MType: models.Gauge,
		Value: &value,
	}
}

func (m *memStorage) UpdateCounter(name string, value int64) {
	if existing, exists := m.data[name]; exists && existing.MType == "counter" {
		newDelta := *existing.Delta + value
		m.data[name] = models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &newDelta,
		}
	} else {
		m.data[name] = models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &value,
		}
	}
}

func (m *memStorage) GetGauge(name string) (float64, bool) {
	if metric, exists := m.data[name]; exists && metric.MType == "gauge" {
		return *metric.Value, true
	}
	return 0, false
}

func (m *memStorage) GetCounter(name string) (int64, bool) {
	if metric, exists := m.data[name]; exists && metric.MType == "counter" {
		return *metric.Delta, true
	}
	return 0, false
}

func (m *memStorage) GetAllMetrics()
