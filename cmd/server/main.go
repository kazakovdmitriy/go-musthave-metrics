package main

import (
	"net/http"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg := config.ParseServerConfig()

	if err := logger.Initialise(cfg.LogLevel); err != nil {
		return err
	}

	handler := handler.SetupHandler()

	logger.Log.Info("Run server on: ", zap.String("address", cfg.ServerAddr))
	return http.ListenAndServe(cfg.ServerAddr, handler)
}
