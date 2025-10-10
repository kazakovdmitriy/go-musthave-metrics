package handler

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/mainpage"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/metrics"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/repository"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
)

func SetupHandler(
	memStorage repository.Storage,
	activeRequests *sync.WaitGroup,
	shutdownChan chan struct{},
) http.Handler {
	r := chi.NewRouter()

	compressorService := service.NewGzipCompressor()

	setupMiddlewares(
		r,
		compressorService,
		activeRequests,
		shutdownChan,
	)

	mainPageHandler := newMainPageService(memStorage)
	setupMainRoutes(r, mainPageHandler)

	metricsHandler := newMetricsHandler(memStorage)
	setupMetricsRoutes(r, metricsHandler)

	return r
}

func setupMiddlewares(
	r chi.Router,
	compressorService middlewares.Compressor,
	activeRequests *sync.WaitGroup,
	shutdownChan chan struct{},
) {
	r.Use(middlewares.RequestLogger)
	r.Use(middlewares.ResponseLogger)
	r.Use(middlewares.TrackActiveRequests(activeRequests, shutdownChan))
	r.Use(middlewares.Compress(compressorService))
}

func newMainPageService(memStorage repository.Storage) *mainpage.MainPageHandler {
	mainPageService := service.NewMainPageService(memStorage)
	return mainpage.NewMainPageHandler(mainPageService)
}

func setupMainRoutes(r chi.Router, mainPageHandler *mainpage.MainPageHandler) {
	r.Route("/", func(r chi.Router) {
		r.Get("/", mainPageHandler.GetMainPage)
	})
}

func newMetricsHandler(memStorage repository.Storage) *metrics.MetricsHandler {
	metricsServer := service.NewMetricService(memStorage)
	return metrics.NewMetricsHandler(metricsServer)
}

func setupMetricsRoutes(r chi.Router, metricsHandler *metrics.MetricsHandler) {
	r.Route("/update", func(r chi.Router) {
		r.Route("/{metricType}/{metricName}", func(r chi.Router) {
			r.Post("/{value}", metricsHandler.UpdateMetric)
		})

		r.Post("/", metricsHandler.UpdatePost)
	})

	r.Route("/value", func(r chi.Router) {
		r.Route("/{metricType}/{metricName}", func(r chi.Router) {
			r.Get("/", metricsHandler.GetMetric)
		})

		r.Post("/", metricsHandler.SentMetricPost)
	})
}
