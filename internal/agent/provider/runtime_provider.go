package provider

import (
	"context"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"math/rand/v2"
	"runtime"
)

// RuntimeMetricsProvider поставщик метрик runtime
type RuntimeMetricsProvider struct{}

// NewRuntimeMetricsProvider создает нового поставщика метрик runtime
func NewRuntimeMetricsProvider() interfaces.MetricsProvider {
	return &RuntimeMetricsProvider{}
}

// Collect собирает метрики runtime
func (p *RuntimeMetricsProvider) Collect(ctx context.Context) (model.MemoryMetrics, error) {
	select {
	case <-ctx.Done():
		return model.MemoryMetrics{}, ctx.Err()
	default:
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		return model.MemoryMetrics{
			Alloc:         float64(m.Alloc),
			BuckHashSys:   float64(m.BuckHashSys),
			Frees:         float64(m.Frees),
			GCCPUFraction: m.GCCPUFraction,
			GCSys:         float64(m.GCSys),
			HeapAlloc:     float64(m.HeapAlloc),
			HeapIdle:      float64(m.HeapIdle),
			HeapInuse:     float64(m.HeapInuse),
			HeapObjects:   float64(m.HeapObjects),
			HeapReleased:  float64(m.HeapReleased),
			HeapSys:       float64(m.HeapSys),
			LastGC:        float64(m.LastGC),
			Lookups:       float64(m.Lookups),
			MCacheInuse:   float64(m.MCacheInuse),
			MCacheSys:     float64(m.MCacheSys),
			MSpanInuse:    float64(m.MSpanInuse),
			MSpanSys:      float64(m.MSpanSys),
			Mallocs:       float64(m.Mallocs),
			NextGC:        float64(m.NextGC),
			NumForcedGC:   float64(m.NumForcedGC),
			NumGC:         float64(m.NumGC),
			OtherSys:      float64(m.OtherSys),
			PauseTotalNs:  float64(m.PauseTotalNs),
			StackInuse:    float64(m.StackInuse),
			StackSys:      float64(m.StackSys),
			Sys:           float64(m.Sys),
			TotalAlloc:    float64(m.TotalAlloc),
			RandomValue:   rand.Float64(),
		}, nil
	}
}
