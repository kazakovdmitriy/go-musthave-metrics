package repository

import (
	"sync"
	"testing"
)

// --- Вариант 1: с обычным Mutex ---
type memStorageMutex struct {
	mu       sync.Mutex
	counters map[string]int64
	gauges   map[string]float64
}

func newMemStorageMutex() *memStorageMutex {
	return &memStorageMutex{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (m *memStorageMutex) UpdateGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *memStorageMutex) UpdateCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *memStorageMutex) GetGauge(name string) (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.gauges[name]
	return v, ok
}

func (m *memStorageMutex) GetCounter(name string) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.counters[name]
	return v, ok
}

// --- Вариант 2: с RWMutex ---
type memStorageRWMutex struct {
	mu       sync.RWMutex
	counters map[string]int64
	gauges   map[string]float64
}

func newMemStorageRWMutex() *memStorageRWMutex {
	return &memStorageRWMutex{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (m *memStorageRWMutex) UpdateGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *memStorageRWMutex) UpdateCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *memStorageRWMutex) GetGauge(name string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.gauges[name]
	return v, ok
}

func (m *memStorageRWMutex) GetCounter(name string) (int64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.counters[name]
	return v, ok
}

// --- Бенчмарки ---

// Read-heavy: 90% чтений, 10% записей
func BenchmarkMutex_ReadHeavy(b *testing.B) {
	runBenchmark(b, newMemStorageMutex(), 9, 1)
}

func BenchmarkRWMutex_ReadHeavy(b *testing.B) {
	runBenchmark(b, newMemStorageRWMutex(), 9, 1)
}

// Balanced: 50% чтений, 50% записей
func BenchmarkMutex_Balanced(b *testing.B) {
	runBenchmark(b, newMemStorageMutex(), 1, 1)
}

func BenchmarkRWMutex_Balanced(b *testing.B) {
	runBenchmark(b, newMemStorageRWMutex(), 1, 1)
}

// Write-heavy: 10% чтений, 90% записей
func BenchmarkMutex_WriteHeavy(b *testing.B) {
	runBenchmark(b, newMemStorageMutex(), 1, 9)
}

func BenchmarkRWMutex_WriteHeavy(b *testing.B) {
	runBenchmark(b, newMemStorageRWMutex(), 1, 9)
}

// Общий runner для всех сценариев
func runBenchmark(b *testing.B, s interface{}, readRatio, writeRatio int) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			counter++
			op := counter % (readRatio + writeRatio)
			if op < readRatio {
				// Чтение
				switch st := s.(type) {
				case *memStorageMutex:
					st.GetGauge("g1")
					st.GetCounter("c1")
				case *memStorageRWMutex:
					st.GetGauge("g1")
					st.GetCounter("c1")
				}
			} else {
				// Запись
				switch st := s.(type) {
				case *memStorageMutex:
					st.UpdateGauge("g1", float64(counter))
					st.UpdateCounter("c1", 1)
				case *memStorageRWMutex:
					st.UpdateGauge("g1", float64(counter))
					st.UpdateCounter("c1", 1)
				}
			}
		}
	})
}

// Подготовка данных перед бенчмарком (чтобы не было паник при чтении пустых ключей)
func init() {
	// Заполним начальные значения для обоих типов
	msm := newMemStorageMutex()
	msm.UpdateGauge("g1", 0)
	msm.UpdateCounter("c1", 0)

	msrw := newMemStorageRWMutex()
	msrw.UpdateGauge("g1", 0)
	msrw.UpdateCounter("c1", 0)
}
