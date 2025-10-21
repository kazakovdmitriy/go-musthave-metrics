package ping

import (
	"context"
	"net/http"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
	"go.uber.org/zap"
)

type PingHandler struct {
	log     *zap.Logger
	storage service.Storage
}

func NewPingHandler(log *zap.Logger, storage service.Storage) *PingHandler {
	return &PingHandler{
		log:     log,
		storage: storage,
	}
}

func (h *PingHandler) GetPingDB(w http.ResponseWriter, r *http.Request) {
	if healthChecker, ok := h.storage.(HealthChecker); ok {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := healthChecker.Ping(ctx); err != nil {
			h.log.Warn("Failed to ping database", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	h.log.Warn("Storage does not support health checks")
	w.WriteHeader(http.StatusInternalServerError)
}
