package model

import "time"

// MetricProcessedEvent - событие обработки метрики
// generate:reset
type MetricProcessedEvent struct {
	Timestamp time.Time `json:"-"`  // Внутреннее представление времени
	TS        int64     `json:"ts"` // Unix timestamp в миллисекундах
	Metrics   []string  `json:"metrics"`
	IPAddr    string    `json:"ip_address"`
}
