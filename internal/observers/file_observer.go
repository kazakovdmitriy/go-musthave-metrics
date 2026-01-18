package observers

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

type FileObserver struct {
	filePath string
	log      *zap.Logger
	mu       sync.Mutex
}

func NewFileObserver(filePath string, log *zap.Logger) *FileObserver {
	return &FileObserver{
		filePath: filePath,
		log:      log,
	}
}

func (f *FileObserver) OnMetricProcessed(event model.MetricProcessedEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		f.log.Error("Error opening file", zap.Error(err), zap.String("file", f.filePath))
		return
	}
	defer file.Close()

	jsonEvent, err := json.Marshal(event)
	if err != nil {
		f.log.Error("Error marshaling event", zap.Error(err), zap.String("file", f.filePath))
		return
	}

	jsonEvent = append(jsonEvent, '\n')

	if _, err := file.Write(jsonEvent); err != nil {
		f.log.Error("Error writing to file", zap.Error(err), zap.String("file", f.filePath))
		return
	}

	if err := file.Sync(); err != nil {
		f.log.Warn("Error syncing file", zap.Error(err), zap.String("file", f.filePath))
	}

	f.log.Debug("Metric processed", zap.Any("event", event))
}
