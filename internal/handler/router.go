package handler

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/mainpage"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/metrics"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares/compressor"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
	"go.uber.org/zap"
)

func SetupHandler(
	memStorage service.Storage,
	activeRequests *sync.WaitGroup,
	log *zap.Logger,
	shutdownChan chan struct{},
) http.Handler {
	r := chi.NewRouter()

	compressorService := compressor.NewHTTPGzipAdapter()

	setupMiddlewares(
		r,
		compressorService,
		activeRequests,
		shutdownChan,
		log,
	)

	mainPageHandler := newMainPageService(memStorage)
	setupMainRoutes(r, mainPageHandler)

	metricsHandler := newMetricsHandler(memStorage, log)
	setupMetricsRoutes(r, metricsHandler)

	return r
}

func setupMiddlewares(
	r chi.Router,
	compressorService compressor.Compressor,
	activeRequests *sync.WaitGroup,
	shutdownChan chan struct{},
	log *zap.Logger,
) {
	r.Use(middlewares.RequestLogger(log))
	r.Use(middlewares.ResponseLogger(log))
	r.Use(middlewares.TrackActiveRequests(activeRequests, shutdownChan))
	r.Use(compressor.Compress(compressorService, log))
}

func newMainPageService(memStorage service.Storage) *mainpage.MainPageHandler {
	mainPageService := service.NewMainPageService(memStorage)
	return mainpage.NewMainPageHandler(mainPageService)
}

func setupMainRoutes(r chi.Router, mainPageHandler *mainpage.MainPageHandler) {
	r.Route("/", func(r chi.Router) {
		r.Get("/", mainPageHandler.GetMainPage)
	})
}

func newMetricsHandler(memStorage service.Storage, log *zap.Logger) *metrics.MetricsHandler {
	metricsServer := service.NewMetricService(memStorage)
	return metrics.NewMetricsHandler(metricsServer, log)
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
