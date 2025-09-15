package agent

import (
	"fmt"
	"reflect"
)

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

func SendMetrics(client *Client, metrics MemoryMetrics) ([]byte, error) {

	metricsMap := metricsToMap(metrics)

	for name, value := range metricsMap {
		_, err := client.Post(fmt.Sprintf("/update/gauge/%s/%f", name, value), nil)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}
