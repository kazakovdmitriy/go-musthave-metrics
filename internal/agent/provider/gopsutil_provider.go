package provider

import (
	"context"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// GopsutilProvider поставщик системных метрик через gopsutil
type GopsutilProvider struct{}

// NewGopsutilProvider создает нового поставщика gopsutil метрик
func NewGopsutilProvider() interfaces.MetricsProvider {
	return &GopsutilProvider{}
}

// Collect собирает системные метрики
func (p *GopsutilProvider) Collect(ctx context.Context) (model.MemoryMetrics, error) {
	select {
	case <-ctx.Done():
		return model.MemoryMetrics{}, ctx.Err()
	default:
		metrics := model.MemoryMetrics{}

		memInfo, err := mem.VirtualMemoryWithContext(ctx)
		if err != nil {
			return metrics, err
		}

		metrics.TotalMemory = float64(memInfo.Total)
		metrics.FreeMemory = float64(memInfo.Free)

		cpuPercent, err := cpu.PercentWithContext(ctx, 0, true)
		if err != nil {
			return metrics, err
		}

		metrics.CPUutilization = cpuPercent

		return metrics, nil
	}
}
