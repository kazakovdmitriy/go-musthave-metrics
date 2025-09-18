package agent

import (
	"fmt"
)

func SendMetrics(client *Client, metrics MemoryMetrics) ([]byte, error) {

	metricsMap := metrics.ToMap()

	for name, value := range metricsMap {
		_, err := client.Post(fmt.Sprintf("/update/gauge/%s/%f", name, value), nil)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}
