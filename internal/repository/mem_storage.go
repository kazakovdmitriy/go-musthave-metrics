package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

type memStorage struct {
	mu       sync.RWMutex
	ticker   *time.Ticker
	counters map[string]int64
	gauges   map[string]float64
	done     chan bool
}

func NewMemStorage(cfg *config.ServerFlags) Storage {

	storage := &memStorage{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}

	if cfg.Restore {
		storage.LoadFromFile(cfg.FileStoragePath)
	}

	return storage
}

func (m *memStorage) UpdateGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *memStorage) UpdateCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, exists := m.counters[name]; exists {
		newDelta := existing + value
		m.counters[name] = newDelta
	} else {
		m.counters[name] = value
	}
}

func (m *memStorage) GetGauge(name string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if metric, exists := m.gauges[name]; exists {
		return metric, true
	}
	return 0, false
}

func (m *memStorage) GetCounter(name string) (int64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if metric, exists := m.counters[name]; exists {
		return metric, true
	}
	return 0, false
}

func (m *memStorage) GetAllMetrics() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

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

func (m *memStorage) SaveToFile(filename string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var metrics []model.Metrics

	for id, delta := range m.counters {
		deltaCopy := delta
		metrics = append(metrics, model.Metrics{
			ID:    id,
			MType: "counter",
			Delta: &deltaCopy,
		})
	}

	for id, value := range m.gauges {
		valueCopy := value
		metrics = append(metrics, model.Metrics{
			ID:    id,
			MType: "gauge",
			Value: &valueCopy,
		})
	}

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (m *memStorage) LoadFromFile(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Log.Error("error while loading file", zap.Error(err))
			return nil
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	var metrics []model.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	for _, metric := range metrics {
		switch metric.MType {
		case "counter":
			if metric.Delta != nil {
				m.counters[metric.ID] = *metric.Delta
			}
		case "gauge":
			if metric.Value != nil {
				m.gauges[metric.ID] = *metric.Value
			}
		}
	}

	logger.Log.Info("load metrics from file", zap.String("filename", filename))

	return nil
}

func (m *memStorage) StartPeriodicSave(interval time.Duration, filename string) {
	m.ticker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-m.ticker.C:
				if err := m.SaveToFile(filename); err != nil {
					logger.Log.Error("Ошибка сохранения: ", zap.Error(err))
				} else {
					logger.Log.Info("Данные сохранены в файл", zap.String("file_name", filename))
				}
			case <-m.done:
				return
			}
		}
	}()
}

func (m *memStorage) StopPeriodicSave() {
	if m.ticker != nil {
		m.ticker.Stop()
	}
	select {
	case m.done <- true:
	default:
	}
}
