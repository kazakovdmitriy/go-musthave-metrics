package main

import (
	"net/http"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/repository"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {

	memStorage := repository.NewMemStorage()
	metricsService := service.NewMetricService(memStorage)
	metricsHandler := handler.NewMetricsHandler(metricsService)

	mux := http.NewServeMux()

	mux.HandleFunc("/update/", metricsHandler.Update)

	return http.ListenAndServe(`:8080`, mux)
}
