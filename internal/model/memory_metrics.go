package model

import "fmt"

// MemoryMetrics представляет метрики памяти
type MemoryMetrics struct {
	Alloc         float64 `json:"alloc"`
	BuckHashSys   float64 `json:"buck_hash_sys"`
	Frees         float64 `json:"frees"`
	GCCPUFraction float64 `json:"gccpu_fraction"`
	GCSys         float64 `json:"gc_sys"`
	HeapAlloc     float64 `json:"heap_alloc"`
	HeapIdle      float64 `json:"heap_idle"`
	HeapInuse     float64 `json:"heap_inuse"`
	HeapObjects   float64 `json:"heap_objects"`
	HeapReleased  float64 `json:"heap_released"`
	HeapSys       float64 `json:"heap_sys"`
	LastGC        float64 `json:"last_gc"`
	Lookups       float64 `json:"lookups"`
	MCacheInuse   float64 `json:"m_cache_inuse"`
	MCacheSys     float64 `json:"m_cache_sys"`
	MSpanInuse    float64 `json:"m_span_inuse"`
	MSpanSys      float64 `json:"m_span_sys"`
	Mallocs       float64 `json:"mallocs"`
	NextGC        float64 `json:"next_gc"`
	NumForcedGC   float64 `json:"num_forced_gc"`
	NumGC         float64 `json:"num_gc"`
	OtherSys      float64 `json:"other_sys"`
	PauseTotalNs  float64 `json:"pause_total_ns"`
	StackInuse    float64 `json:"stack_inuse"`
	StackSys      float64 `json:"stack_sys"`
	Sys           float64 `json:"sys"`
	TotalAlloc    float64 `json:"total_alloc"`
	RandomValue   float64 `json:"random_value,omitempty"`

	TotalMemory    float64   `json:"total_memory"`
	FreeMemory     float64   `json:"free_memory"`
	CPUutilization []float64 `json:"cpu_utilization"`
}

// ToMap преобразует метрики в map
func (m MemoryMetrics) ToMap() map[string]float64 {
	result := map[string]float64{
		"Alloc":         m.Alloc,
		"BuckHashSys":   m.BuckHashSys,
		"Frees":         m.Frees,
		"GCCPUFraction": m.GCCPUFraction,
		"GCSys":         m.GCSys,
		"HeapAlloc":     m.HeapAlloc,
		"HeapIdle":      m.HeapIdle,
		"HeapInuse":     m.HeapInuse,
		"HeapObjects":   m.HeapObjects,
		"HeapReleased":  m.HeapReleased,
		"HeapSys":       m.HeapSys,
		"LastGC":        m.LastGC,
		"Lookups":       m.Lookups,
		"MCacheInuse":   m.MCacheInuse,
		"MCacheSys":     m.MCacheSys,
		"MSpanInuse":    m.MSpanInuse,
		"MSpanSys":      m.MSpanSys,
		"Mallocs":       m.Mallocs,
		"NextGC":        m.NextGC,
		"NumForcedGC":   m.NumForcedGC,
		"NumGC":         m.NumGC,
		"OtherSys":      m.OtherSys,
		"PauseTotalNs":  m.PauseTotalNs,
		"StackInuse":    m.StackInuse,
		"StackSys":      m.StackSys,
		"Sys":           m.Sys,
		"TotalAlloc":    m.TotalAlloc,
		"RandomValue":   m.RandomValue,
		"TotalMemory":   m.TotalMemory,
		"FreeMemory":    m.FreeMemory,
	}

	// Добавляем метрики CPU
	for i, utilization := range m.CPUutilization {
		result[fmt.Sprintf("CPUutilization%d", i+1)] = utilization
	}

	return result
}

// String возвращает строковое представление метрик
func (m MemoryMetrics) String() string {
	return fmt.Sprintf("MemoryMetrics{Alloc: %.2f, HeapAlloc: %.2f, Sys: %.2f}",
		m.Alloc, m.HeapAlloc, m.Sys)
}

// IsZero проверяет являются ли метрики нулевыми
func (m MemoryMetrics) IsZero() bool {
	return m.Alloc == 0 && m.Sys == 0 && m.TotalAlloc == 0
}

// Clone создает глубокую копию метрик
func (m MemoryMetrics) Clone() MemoryMetrics {
	clone := m
	if m.CPUutilization != nil {
		clone.CPUutilization = make([]float64, len(m.CPUutilization))
		copy(clone.CPUutilization, m.CPUutilization)
	}
	return clone
}
