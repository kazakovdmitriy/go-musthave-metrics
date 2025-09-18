package agent

import (
	"fmt"
	"reflect"
	"runtime"
)

func getMetric() MemoryMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return MemoryMetrics{
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
	}
}

func metricsToMap(metrics MemoryMetrics) map[string]float64 {
	result := make(map[string]float64)

	v := reflect.ValueOf(metrics)
	t := reflect.TypeOf(metrics)

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i).Float()
		result[field.Name] = value
	}

	return result
}

func SendMetrics(client *Client) ([]byte, error) {

	metrics := getMetric()
	metricsMap := metricsToMap(metrics)

	for name, value := range metricsMap {
		_, err := client.Post(fmt.Sprintf("/gauge/%s/%f", name, value), nil)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}
