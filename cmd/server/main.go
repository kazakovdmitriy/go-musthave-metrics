package main

import (
	"net/http"

	"github.com/go-chi/chi"
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

	r := chi.NewRouter()

	r.Route("/update", func(r chi.Router) {
		r.Route("/{metricType}/{metricName}", func(r chi.Router) {
			r.Get("/", metricsHandler.GetMetric)
			r.Post("/{value}", metricsHandler.UpdateMetric)
		})
	})

	return r
}
