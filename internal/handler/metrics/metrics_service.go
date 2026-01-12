package metrics

import (
	"context"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
)

// MetricsService определяет интерфейс для взаимодействия с хранилищем метрик.
// Должен быть реализован сервисом, инкапсулирующим логику получения и обновления метрик.
type MetricsService interface {
	// UpdateGauge устанавливает новое значение для метрики типа gauge.
	UpdateGauge(ctx context.Context, name string, value float64) error

	// UpdateCounter увеличивает значение метрики типа counter на указанную дельту.
	UpdateCounter(ctx context.Context, name string, value int64) error

	// UpdateMetrics обновляет несколько метрик за один вызов (пакетное обновление).
	UpdateMetrics(ctx context.Context, metrics []model.Metrics, ipAddr string) error

	// GetGauge возвращает текущее значение метрики типа gauge по её имени.
	GetGauge(ctx context.Context, name string) (float64, error)

	// GetCounter возвращает текущее значение метрики типа counter по её имени.
	GetCounter(ctx context.Context, name string) (int64, error)
}
