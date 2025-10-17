package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/repository"
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
	storage := repository.NewMemStorage(cfg, log)
	app := &Server{
		cfg:     cfg,
		log:     log,
		storage: storage,
	}
	return app, nil
}

func (a *Server) Run() error {
	var activeRequests sync.WaitGroup
	shutdownCh := make(chan struct{})

	handler := handler.SetupHandler(
		a.storage,
		&activeRequests,
		a.log,
		shutdownCh,
	)

	a.server = &http.Server{
		Addr:    a.cfg.ServerAddr,
		Handler: handler,
	}

	go func() {
		a.log.Info("server starting", zap.String("addr", a.cfg.ServerAddr))
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.log.Error("server failed to start", zap.Error(err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
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

	a.log.Info("server stopped gracefully")
	return nil
}

func (a *Server) Close() {
	if a.storage != nil {
		a.storage.Close()
	}
}
