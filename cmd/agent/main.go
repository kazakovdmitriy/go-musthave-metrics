package main

import (
	"fmt"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
)

func main() {

	cfg := config.ParseFlagsAgent()
	client := agent.NewClient(cfg.ServerAddr)

	poolingInterval := time.Duration(cfg.PollingInterval) * time.Second
	reportInterval := time.Duration(cfg.ReportInterval) * time.Second

	metrics := agent.GetMetrics()
	lastReportTime := time.Now()

	for {

		currentTime := time.Now()

		if currentTime.Sub(lastReportTime) >= reportInterval {
			_, err := agent.SendMetrics(client, metrics)
			if err != nil {
				fmt.Println("error from server: ", err)
			}
		}

		metrics = agent.GetMetrics()
		time.Sleep(poolingInterval)
	}
}
