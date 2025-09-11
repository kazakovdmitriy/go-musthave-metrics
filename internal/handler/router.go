package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/repository"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
)

func SetupHandler() http.Handler {
	r := chi.NewRouter()

	memStorage := repository.NewMemStorage()

	mainPageHandler := newMainPageService(memStorage)
	setupMainRoutes(r, mainPageHandler)

	metricsHandler := newMetricsHandler(memStorage)
	setupMetricsRoutes(r, metricsHandler)

	return r
}

func newMainPageService(memStorage repository.Storage) *MainPageHandler {
	mainPageService := service.NewMainPageService(memStorage)
	return NewMainPageHandler(mainPageService)
}

func setupMainRoutes(r chi.Router, mainPageHandler *MainPageHandler) {
	r.Route("/", func(r chi.Router) {
		r.Get("/", mainPageHandler.GetMainPage)
	})
}

func newMetricsHandler(memStorage repository.Storage) *MetricsHandler {
	metricsServer := service.NewMetricService(memStorage)
	return NewMetricsHandler(metricsServer)
}

func setupMetricsRoutes(r chi.Router, metricsHandler *MetricsHandler) {
	r.Route("/update", func(r chi.Router) {
		r.Route("/{metricType}/{metricName}", func(r chi.Router) {
			r.Get("/", metricsHandler.GetMetric)
			r.Post("/{value}", metricsHandler.UpdateMetric)
		})
	})
}
