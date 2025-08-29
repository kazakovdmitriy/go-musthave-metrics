package main

import (
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent"
)

func main() {
	for {
		metrics := agent.GetMetric()
		metricsMap := agent.MetricsToMap(metrics)

		agent.SendMetrics(metricsMap)

		time.Sleep(2 * time.Second)
	}
}
