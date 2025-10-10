package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/repository"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		logger.Log.Error("application error", zap.Error(err))
		os.Exit(1)
	}
}

func run() error {
	cfg := config.ParseServerConfig()

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	memStorage := repository.NewMemStorage(cfg)

	if cfg.StoreInterval > 0 {
		stopBackup := repository.StartStorageBackup(
			memStorage,
			time.Duration(cfg.StoreInterval)*time.Second,
			cfg.FileStoragePath,
		)
		defer stopBackup()
	}

	var activeRequests sync.WaitGroup
	shutdownChan := make(chan struct{})

	handler := handler.SetupHandler(
		memStorage,
		&activeRequests,
		shutdownChan,
	)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: handler,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Log.Info("run server on: ", zap.String("address", cfg.ServerAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-interrupt:
		logger.Log.Info("received shutdown signal", zap.String("signal", sig.String()))
	}

	logger.Log.Info("starting graceful shutdown...")
	close(shutdownChan)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("server shutdown error", zap.Error(err))
	}

	logger.Log.Info("waiting for active requests to complete...")

	waitDone := make(chan struct{})
	go func() {
		activeRequests.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		logger.Log.Info("all active requests completed")
	case <-time.After(10 * time.Second):
		logger.Log.Warn("timeout waiting for active requests")
	}

	logger.Log.Info("Server stopped gracefully")
	return nil
}
