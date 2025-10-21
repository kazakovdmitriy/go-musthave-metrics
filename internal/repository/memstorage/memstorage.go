package memstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
	"go.uber.org/zap"
)

type memStorage struct {
	mu       *sync.Mutex
	counters map[string]int64
	gauges   map[string]float64
	cfg      *config.ServerFlags
	log      *zap.Logger

	tickerMu *sync.Mutex
	ticker   *time.Ticker
	done     chan struct{}
}

func NewMemStorage(cfg *config.ServerFlags, log *zap.Logger) service.Storage {

	storage := &memStorage{
		mu:       &sync.Mutex{},
		tickerMu: &sync.Mutex{},
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
		cfg:      cfg,
		done:     make(chan struct{}),
		log:      log,
	}

	if cfg.Restore {
		storage.LoadFromFile(cfg.FileStoragePath)
	}

	if cfg.StoreInterval > 0 {
		storage.StartPeriodicSave(
			time.Duration(cfg.StoreInterval)*time.Second,
			cfg.FileStoragePath,
		)
	}

	return storage
}

func (m *memStorage) UpdateGauge(ctx context.Context, name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *memStorage) UpdateCounter(ctx context.Context, name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, exists := m.counters[name]; exists {
		newDelta := existing + value
		m.counters[name] = newDelta
	} else {
		m.counters[name] = value
	}
}

func (m *memStorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if metric, exists := m.gauges[name]; exists {
		return metric, true
	}
	return 0, false
}

func (m *memStorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if metric, exists := m.counters[name]; exists {
		return metric, true
	}
	return 0, false
}

func (m *memStorage) GetAllMetrics(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result string

	result += "<ul>\n"

	for key, value := range m.counters {
		result += fmt.Sprintf("<li>%s = %d</li>\n", key, value)
	}

	for key, value := range m.gauges {
		result += fmt.Sprintf("<li>%s = %f</li>\n", key, value)
	}

	result += "</ul>\n"

	if result != "<ul>\n</ul>\n" {
		return result, nil
	}

	return "", fmt.Errorf("no metrics found")
}

func (m *memStorage) SaveToFile(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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
			m.log.Error("error while loading file", zap.Error(err))
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

	m.log.Info("load metrics from file", zap.String("filename", filename))

	return nil
}

func (m *memStorage) StartPeriodicSave(interval time.Duration, filename string) {
	m.tickerMu.Lock()
	defer m.tickerMu.Unlock()

	if m.ticker != nil {
		m.log.Warn("periodic save already started")
		return
	}

	m.ticker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-m.ticker.C:
				if err := m.SaveToFile(filename); err != nil {
					m.log.Error("failed to save: ", zap.Error(err))
				} else {
					m.log.Info("metrics save to file", zap.String("file_name", filename))
				}
			case <-m.done:
				m.log.Info("periodic save stopped")
				return
			}
		}
	}()
}

func (m *memStorage) Close() error {
	m.tickerMu.Lock()
	defer m.tickerMu.Unlock()

	if m.ticker != nil {
		m.ticker.Stop()
		m.ticker = nil
	}

	select {
	case <-m.done:
	default:
		close(m.done)
	}

	if err := m.SaveToFile(m.cfg.FileStoragePath); err != nil {
		return fmt.Errorf("failed to save on close: %w", err)
	}

	return nil
}
