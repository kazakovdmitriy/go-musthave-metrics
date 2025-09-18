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
	handler := setupHandler()
	return http.ListenAndServe(":8080", handler)
}

func setupHandler() http.Handler {
	memStorage := repository.NewMemStorage()
	metricsServer := service.NewMetricService(memStorage)
	metricsHandler := handler.NewMetricsHandler(metricsServer)

	mux := http.NewServeMux()

	mux.HandleFunc("/update/", metricsHandler.Update)

	return mux
}
