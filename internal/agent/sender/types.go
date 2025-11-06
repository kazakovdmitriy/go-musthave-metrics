package sender

import "github.com/kazakovdmitriy/go-musthave-metrics/internal/model"

// SendTask задача для отправки метрик
type SendTask struct {
	Metrics model.MemoryMetrics
	Count   int64
}
