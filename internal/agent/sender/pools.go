package sender

import (
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"github.com/kazakovdmitriy/go-musthave-metrics/pkg/objpool"
)

// MetricsSlice - обертка для слайса метрик с методом Reset
type MetricsSlice struct {
	Slice []model.Metrics
}

// Reset очищает слайс, сохраняя capacity
func (m *MetricsSlice) Reset() {
	m.Slice = m.Slice[:0]
}

// MetricsBatchPool - пул для батчей метрик
type MetricsBatchPool struct {
	pool *objpool.Pool[*MetricsSlice]
}

// NewMetricsBatchPool создает пул для батчей метрик
func NewMetricsBatchPool(initialCapacity int) *MetricsBatchPool {
	return &MetricsBatchPool{
		pool: objpool.New(func() *MetricsSlice {
			return &MetricsSlice{
				Slice: make([]model.Metrics, 0, initialCapacity),
			}
		}),
	}
}

// GetBatch возвращает батч из пула
func (p *MetricsBatchPool) GetBatch() *MetricsSlice {
	return p.pool.Get()
}

// PutBatch возвращает батч в пул
func (p *MetricsBatchPool) PutBatch(batch *MetricsSlice) {
	p.pool.Put(batch)
}
