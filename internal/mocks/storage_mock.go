package mocks

import (
	"context"
)

type MockStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *MockStorage) UpdateGauge(ctx context.Context, name string, value float64) {
	m.gauges[name] = value
}

func (m *MockStorage) UpdateCounter(ctx context.Context, name string, value int64) {
	m.counters[name] += value
}

func (m *MockStorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	val, ok := m.gauges[name]
	return val, ok
}

func (m *MockStorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	val, ok := m.counters[name]
	return val, ok
}

func (m *MockStorage) GetAllMetrics(ctx context.Context) (string, error) {
	return "", nil
}

func (m *MockStorage) Close() error {
	return nil
}
