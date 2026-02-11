package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/mainpagehandler"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/metricshandler"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares/compressor"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares/signer"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/pinghandler"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/observers"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service/mainpageservice"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service/metricsservice"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service/signerservice"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/utils/config"
	db2 "github.com/kazakovdmitriy/go-musthave-metrics/internal/utils/config/db"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/repository/dbstorage"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/repository/memstorage"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
	"go.uber.org/zap"
)

type Server struct {
	cfg     *config.ServerFlags
	log     *zap.Logger
	storage service.Storage
	server  *http.Server
}

func NewApp(cfg *config.ServerFlags, log *zap.Logger) (*Server, error) {
	app := &Server{
		cfg: cfg,
		log: log,
	}
	return app, nil
}

func (s *Server) Run(ctx context.Context) error {
	// 1. Инициализируем все зависимости
	storage, subject, resources, err := s.initDependencies(ctx)
	if err != nil {
		return fmt.Errorf("failed to init dependencies: %w", err)
	}

	// 2. Создаем WaitGroup для активных запросов
	activeRequests := &sync.WaitGroup{}
	shutdownCh := make(chan struct{})

	// 3. Создаем роутер (все в одном месте)
	router, err := s.createRouter(storage, subject, activeRequests, shutdownCh)
	if err != nil {
		return fmt.Errorf("failed to create router: %w", err)
	}

	// 4. Создаем HTTP сервер
	s.server = &http.Server{
		Addr:    s.cfg.ServerAddr,
		Handler: router,
	}

	// 5. Запускаем сервер
	go func() {
		s.log.Info("server starting", zap.String("addr", s.cfg.ServerAddr))
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("server failed to start", zap.Error(err))
		}
	}()

	// 6. Graceful shutdown
	return s.waitForShutdown(ctx, resources, activeRequests, shutdownCh)
}

func (s *Server) Close() {
	if s.storage != nil {
		if err := s.storage.Close(); err != nil {
			s.log.Error("storage close failed", zap.Error(err))
		}
	}
}

func (s *Server) initDependencies(ctx context.Context) (
	service.Storage,
	metricsservice.EventPublisher,
	[]closableResource,
	error,
) {
	var resources []closableResource

	// Инициализация хранилища
	storage, err := s.initStorage(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("stirage %w", err)
	}
	resources = append(resources, storage)

	// Инициализация наблюдателей
	subject, observerResources, err := s.initObservers()
	if err != nil {
		return nil, nil, resources, fmt.Errorf("observers: %w", err)
	}
	resources = append(resources, observerResources...)

	return storage, subject, resources, nil

}

func (s *Server) initStorage(ctx context.Context) (service.Storage, error) {
	if s.cfg.DatabaseDSN == "" {
		s.log.Info("using in-memory storage")
		return memstorage.NewMemStorage(s.cfg, s.log), nil
	}

	// Пробуем подключиться к БД
	dbase, err := db2.NewDatabase(ctx, s.cfg.DatabaseDSN)
	if err != nil || !dbase.IsConnected() {
		s.log.Warn("database connection failed, falling back to memory", zap.Error(err))
		return memstorage.NewMemStorage(s.cfg, s.log), nil
	}

	// Миграции
	migrator := db2.NewMigrator(s.cfg.DatabaseDSN, "migrations", s.log)
	if err := migrator.Up(); err != nil {
		s.log.Error("migration failed", zap.Error(err))
	}

	// Хранилище БД
	storage, err := dbstorage.NewDBStorage(dbase.Pool, s.log, s.cfg)
	if err != nil {
		s.log.Warn("DB storage init failed, falling back to memory", zap.Error(err))
		return memstorage.NewMemStorage(s.cfg, s.log), nil
	}

	return storage, nil
}

func (s *Server) initObservers() (
	metricsservice.EventPublisher,
	[]closableResource,
	error,
) {
	subject := observers.NewEventPublisher()
	var resources []closableResource

	// Логирующий наблюдатель
	loggerObserver := observers.NewMetricLogger(s.log)
	subject.Register(loggerObserver)

	// Файловый наблюдатель
	if s.cfg.AuditFile != "" {
		fileObserver, err := observers.NewFileObserver(s.cfg.AuditFile, s.log)
		if err != nil {
			return nil, resources, fmt.Errorf("file observer: %w", err)
		}
		subject.Register(fileObserver)
		resources = append(resources, fileObserver)
	}

	// HTTP наблюдатель
	if s.cfg.AuditURL != "" {
		httpObserver, err := observers.NewHTTPObserver(s.cfg.AuditURL, s.log, s.cfg)
		if err != nil {
			return nil, resources, fmt.Errorf("HTTP observer: %w", err)
		}
		subject.Register(httpObserver)
		resources = append(resources, httpObserver)
	}

	return subject, resources, nil
}

// createRouter создает и настраивает роутер со всеми middleware и хендлерами
func (s *Server) createRouter(
	storage service.Storage,
	subject metricsservice.EventPublisher,
	activeRequests *sync.WaitGroup,
	shutdownCh chan struct{},
) (http.Handler, error) {
	r := chi.NewRouter()

	// MIDDLEWARE: Создаем сервисы для middleware
	compressorService := compressor.NewHTTPGzipAdapter()

	var signerService signer.Signer
	if s.cfg.SecretKet != "" {
		signerService = signerservice.NewSHA256Signer(s.cfg.SecretKet)
	}

	decryptor, err := middlewares.NewDecryptor(s.cfg.CryptoKeyPath, s.log)
	if err != nil {
		s.log.Fatal("failed to initialize decryptor", zap.Error(err))
	}

	// MIDDLEWARE: Устанавливаем middleware
	r.Use(middlewares.RequestLogger(s.log))
	r.Use(middlewares.ResponseLogger(s.log))
	r.Use(middlewares.TrackActiveRequests(activeRequests, shutdownCh))
	r.Use(middlewares.RateLimiter(s.cfg.RateLimit, s.log))
	r.Use(decryptor.Middleware)
	r.Use(compressor.Compress(compressorService, s.log))

	if signerService != nil {
		r.Use(signer.HashValidationMiddleware(signerService, s.log))
	}

	// HANDLERS: Создаем сервисы и хендлеры
	mainPageService, err := mainpageservice.NewMainPageService(storage)
	if err != nil {
		return nil, fmt.Errorf("main page service: %w", err)
	}

	metricsService := metricsservice.NewMetricService(storage, subject)

	pingHandler := pinghandler.NewPingHandler(s.log, storage)
	mainPageHandler := mainpagehandler.NewMainPageHandler(mainPageService)
	metricsHandler := metricshandler.NewMetricsHandler(metricsService, s.log)

	// ROUTES: Настраиваем все маршруты
	r.Route("/pinghandler", func(r chi.Router) {
		r.Get("/", pingHandler.GetPingDB)
	})

	r.Route("/", func(r chi.Router) {
		r.Get("/", mainPageHandler.GetMainPage)
	})

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

	return r, nil
}

func (s *Server) waitForShutdown(
	ctx context.Context,
	resources []closableResource,
	activeRequests *sync.WaitGroup,
	shutdownCh chan struct{},
) error {
	ctx, stop := signal.NotifyContext(ctx,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)
	defer stop()

	<-ctx.Done()

	s.log.Info("graceful shutdown initiated")
	close(shutdownCh)

	// Останавливаем HTTP сервер
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.log.Error("server shutdown failed", zap.Error(err))
	}

	// Ждем завершения запросов
	s.log.Info("waiting for active requests...")
	waitDone := make(chan struct{})
	go func() {
		activeRequests.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		s.log.Info("all requests completed")
	case <-time.After(10 * time.Second):
		s.log.Warn("timeout waiting for requests")
	}

	// Закрываем ресурсы
	s.log.Info("closing resources...")
	for _, resource := range resources {
		if err := resource.Close(); err != nil {
			s.log.Error("resource close error", zap.Error(err))
		}
	}

	s.log.Info("server stopped")
	return nil
}

// Интерфейс для ресурсов, которые нужно закрыть
type closableResource interface {
	Close() error
}
