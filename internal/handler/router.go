package handler

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/pprof"
	"sync"

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
	storage service.Storage,
	metricsSubject metricsservice.EventPublisher,
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
		cfg.RateLimit,
		log,
	)

	pingHandler, err := newPingHandler(log, storage)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	setupPingRoutes(r, pingHandler)

	mainPageHandler, err := newMainPageService(storage)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	setupMainRoutes(r, mainPageHandler)

	metricsHandler, err := newMetricsHandler(storage, metricsSubject, log)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	setupMetricsRoutes(r, metricsHandler)

	return r, nil
}

func setupMiddlewares(
	r chi.Router,
	compressorService compressor.Compressor,
	signerService signer.Signer,
	activeRequests *sync.WaitGroup,
	shutdownChan chan struct{},
	maxConcurrent int,
	log *zap.Logger,
) {
	r.Use(middlewares.RequestLogger(log))
	r.Use(middlewares.ResponseLogger(log))
	r.Use(middlewares.TrackActiveRequests(activeRequests, shutdownChan))
	r.Use(middlewares.RateLimiter(maxConcurrent, log))
	r.Use(compressor.Compress(compressorService, log))
	if signerService != nil {
		r.Use(signer.HashValidationMiddleware(signerService, log))
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
func newMainPageService(memStorage service.Storage) (*mainpage.MainPageHandler, error) {
	mainPageService, err := mainpageservice.NewMainPageService(memStorage)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return mainpage.NewMainPageHandler(mainPageService), nil
}

func setupMainRoutes(r chi.Router, mainPageHandler *mainpage.MainPageHandler) {
	r.Route("/", func(r chi.Router) {
		r.Get("/", mainPageHandler.GetMainPage)
	})
}

// Metric
func newMetricsHandler(
	storage service.Storage,
	metricsSubject metricsservice.EventPublisher,
	log *zap.Logger,
) (*metrics.MetricsHandler, error) {
	metricsService := metricsservice.NewMetricService(storage, metricsSubject)
	return metrics.NewMetricsHandler(metricsService, log), nil
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

func pprofRoutes() chi.Router {
	r := chi.NewRouter()

	// Используем Handle вместо Get для обработчиков, которые возвращают http.Handler
	r.HandleFunc("/", pprof.Index)
	r.HandleFunc("/cmdline", pprof.Cmdline)
	r.HandleFunc("/profile", pprof.Profile)
	r.HandleFunc("/symbol", pprof.Symbol)
	r.HandleFunc("/trace", pprof.Trace)

	// Для остальных используем Handle с pprof.Handler
	r.Handle("/heap", pprof.Handler("heap"))
	r.Handle("/goroutine", pprof.Handler("goroutine"))
	r.Handle("/allocs", pprof.Handler("allocs"))
	r.Handle("/block", pprof.Handler("block"))
	r.Handle("/mutex", pprof.Handler("mutex"))
	r.Handle("/threadcreate", pprof.Handler("threadcreate"))

	return r
}
