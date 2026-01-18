package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/observers"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config/db"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler"
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

func (a *Server) Run() error {
	ctx := context.Background()

	resourceGroup := NewResourceGroup(a.log)

	var activeRequests sync.WaitGroup
	shutdownCh := make(chan struct{})

	storage := a.storageInitializer(ctx)
	resourceGroup.Register(storage)

	metricsSubject := observers.NewEventPublisher()

	loggerObserver := observers.NewMetricLogger(a.log)
	metricsSubject.Register(loggerObserver)

	if a.cfg.AuditFile != "" {
		fileObserver, err := observers.NewFileObserver(a.cfg.AuditFile, a.log)
		resourceGroup.Register(fileObserver)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		metricsSubject.Register(fileObserver)
	}

	if a.cfg.AuditURL != "" {
		httpObserver, err := observers.NewHTTPObserver(a.cfg.AuditURL, a.log, a.cfg)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		metricsSubject.Register(httpObserver)
		resourceGroup.Register(httpObserver)
	}

	router, err := handler.SetupHandler(
		storage,
		metricsSubject,
		&activeRequests,
		a.log,
		shutdownCh,
		*a.cfg,
	)
	if err != nil {
		return fmt.Errorf("router initialization error: %w", err)
	}

	a.server = &http.Server{
		Addr:    a.cfg.ServerAddr,
		Handler: router,
	}

	go func() {
		a.log.Info("server starting", zap.String("addr", a.cfg.ServerAddr))
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.log.Error("server failed to start", zap.Error(err))
		}
	}()

	ctx, stop := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)
	defer stop()

	<-ctx.Done()

	a.log.Info("graceful shutdown initiated")
	close(shutdownCh)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.log.Error("server shutdown failed", zap.Error(err))
	}

	a.log.Info("waiting for active requests to complete...")
	waitDone := make(chan struct{})
	go func() {
		activeRequests.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		a.log.Info("all requests completed")
	case <-time.After(10 * time.Second):
		a.log.Warn("timeout waiting for requests")
	}

	a.log.Info("closing all resources...")
	if err := resourceGroup.CloseAll(); err != nil {
		a.log.Error("resource group close failed", zap.Error(err))
	}

	a.log.Info("server stopped gracefully")
	return nil
}

func (a *Server) Close() {
	if a.storage != nil {
		if err := a.storage.Close(); err != nil {
			a.log.Error("storage close failed", zap.Error(err))
		}
	}
}

func (a *Server) storageInitializer(ctx context.Context) service.Storage {
	var storage service.Storage

	if a.cfg.DatabaseDSN != "" {
		dbase, err := db.NewDatabase(ctx, a.cfg.DatabaseDSN)
		if err != nil {
			a.log.Warn("Failed to connect to DB, falling back to in-memory storage", zap.Error(err))
			storage = memstorage.NewMemStorage(a.cfg, a.log)
		} else if !dbase.IsConnected() {
			a.log.Warn("DB connection is not active, falling back to in-memory storage")
			storage = memstorage.NewMemStorage(a.cfg, a.log)
		} else {
			migrator := db.NewMigrator(a.cfg.DatabaseDSN, "migrations", a.log)
			if err := migrator.Up(); err != nil {
				a.log.Error("migration failed", zap.Error(err))
			}
			storage, err = dbstorage.NewDBStorage(dbase.Pool, a.log, a.cfg)
			if err != nil {
				a.log.Warn("Failed to connect to DB, falling back to in-memory storage", zap.Error(err))
			}
		}
	} else {
		a.log.Info("No database DSN provided, using in-memory storage")
		storage = memstorage.NewMemStorage(a.cfg, a.log)
	}

	return storage
}
