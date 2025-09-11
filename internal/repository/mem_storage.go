package repository

import (
	"fmt"
)

type memStorage struct {
	counters map[string]int64
	gauges   map[string]float64
}

func NewMemStorage() Storage {
	return &memStorage{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (m *memStorage) UpdateGauge(name string, value float64) {
	m.gauges[name] = value
}

func (m *memStorage) UpdateCounter(name string, value int64) {
	if existing, exists := m.counters[name]; exists {
		newDelta := existing + value
		m.counters[name] = newDelta
	} else {
		m.counters[name] = value
	}
}

func (m *memStorage) GetGauge(name string) (float64, bool) {
	if metric, exists := m.gauges[name]; exists {
		return metric, true
	}
	return 0, false
}

func (m *memStorage) GetCounter(name string) (int64, bool) {
	if metric, exists := m.counters[name]; exists {
		return metric, true
	}
	return 0, false
}

func (m *memStorage) GetAllMetrics() (string, error) {
	var result string

	result += "<ul>\n"

	for key, value := range m.counters {
		result += fmt.Sprintf("<li>%s = %d</li>\n", key, value)
	}

	for key, value := range m.gauges {
		result += fmt.Sprintf("<li>%s = %f</li>\n", key, value)
	}

	result += "</ul>\n"

	if result != "<ul>\n</ul>\n" { // Проверяем, есть ли данные
		return result, nil
	}

	return "", fmt.Errorf("no metrics found")
}
