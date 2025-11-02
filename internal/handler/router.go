package handler

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/mainpage"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/metrics"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares/compressor"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares/signer"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/ping"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service/mainpageservice"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service/metricsservice"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service/signerservice"
	"go.uber.org/zap"
)

func SetupHandler(
	storage *service.Storage,
	activeRequests *sync.WaitGroup,
	log *zap.Logger,
	shutdownChan chan struct{},
	cfg config.ServerFlags,
) (http.Handler, error) {
	r := chi.NewRouter()

	compressorService := compressor.NewHTTPGzipAdapter()
	var signerService signer.Signer = nil
	if cfg.SecretKet != "" {
		signerService = signerservice.NewSHA256Signer(cfg.SecretKet)
	}

	setupMiddlewares(
		r,
		compressorService,
		signerService,
		activeRequests,
		shutdownChan,
		log,
	)

	pingHandler, err := newPingHandler(log, *storage)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	setupPingRoutes(r, pingHandler)

	mainPageHandler := newMainPageService(*storage)
	setupMainRoutes(r, mainPageHandler)

	metricsHandler := newMetricsHandler(*storage, log)
	setupMetricsRoutes(r, metricsHandler)

	return r, nil
}

func setupMiddlewares(
	r chi.Router,
	compressorService compressor.Compressor,
	signerService signer.Signer,
	activeRequests *sync.WaitGroup,
	shutdownChan chan struct{},
	log *zap.Logger,
) {
	r.Use(middlewares.RequestLogger(log))
	r.Use(middlewares.ResponseLogger(log))
	r.Use(middlewares.TrackActiveRequests(activeRequests, shutdownChan))
	r.Use(compressor.Compress(compressorService, log))

	if signerService != nil {
		r.Use(signer.HashResponseMiddleware(signerService))
		r.Use(signer.HashValidationMiddleware(signerService))
	}
}

// Ping
func newPingHandler(log *zap.Logger, storage service.Storage) (*ping.PingHandler, error) {
	return ping.NewPingHandler(log, storage), nil
}

func setupPingRoutes(r chi.Router, pingHandler *ping.PingHandler) {
	r.Route("/ping", func(r chi.Router) {
		r.Get("/", pingHandler.GetPingDB)
	})
}

// MainPage
func newMainPageService(memStorage service.Storage) *mainpage.MainPageHandler {
	mainPageService := mainpageservice.NewMainPageService(memStorage)
	return mainpage.NewMainPageHandler(mainPageService)
}

func setupMainRoutes(r chi.Router, mainPageHandler *mainpage.MainPageHandler) {
	r.Route("/", func(r chi.Router) {
		r.Get("/", mainPageHandler.GetMainPage)
	})
}

// Metric
func newMetricsHandler(storage service.Storage, log *zap.Logger) *metrics.MetricsHandler {
	metricsService := metricsservice.NewMetricService(storage)
	return metrics.NewMetricsHandler(metricsService, log)
}

func setupMetricsRoutes(r chi.Router, metricsHandler *metrics.MetricsHandler) {
	r.Route("/update", func(r chi.Router) {
		r.Route("/{metricType}/{metricName}", func(r chi.Router) {
			r.Post("/{value}", metricsHandler.UpdateMetric)
		})

		r.Post("/", metricsHandler.UpdatePost)
	})

	r.Route("/updates", func(r chi.Router) {
		r.Post("/", metricsHandler.UpdateMetrics)
	})

	r.Route("/value", func(r chi.Router) {
		r.Route("/{metricType}/{metricName}", func(r chi.Router) {
			r.Get("/", metricsHandler.GetMetric)
		})

		r.Post("/", metricsHandler.SentMetricPost)
	})
}
