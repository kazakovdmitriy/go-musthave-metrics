package model

import "time"

// MetricProcessedEvent - событие обработки метрики
type MetricProcessedEvent struct {
	Timestamp time.Time `json:"-"`  // Внутреннее представление времени
	Ts        int64     `json:"ts"` // Unix timestamp в миллисекундах
	Metrics   []string  `json:"metrics"`
	IpAddr    string    `json:"ip_address"`
}
