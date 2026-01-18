package observers

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

type FileObserver struct {
	file        *os.File
	filePath    string
	log         *zap.Logger
	syncCounter int
	mu          sync.Mutex
}

func NewFileObserver(filePath string, log *zap.Logger) (*FileObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &FileObserver{
		file:     file,
		filePath: filePath,
		log:      log,
	}, nil
}

// Close закрывает файл при завершении работы
func (f *FileObserver) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file == nil {
		return nil
	}

	if err := f.file.Sync(); err != nil {
		f.log.Warn("Sync failed on close", zap.Error(err))
	}

	if err := f.file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	f.file = nil
	f.log.Info("File observer closed", zap.String("path", f.filePath))
	return nil
}

func (f *FileObserver) OnMetricProcessed(event model.MetricProcessedEvent) {
	jsonEvent, err := json.Marshal(event)
	if err != nil {
		f.log.Error("Error marshaling event", zap.Error(err))
		return
	}
	jsonEvent = append(jsonEvent, '\n')

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, err := f.file.Write(jsonEvent); err != nil {
		f.log.Error("Error writing to file", zap.Error(err))
		return
	}

	f.log.Debug("Metric processed", zap.Any("event", event))
}
