// Package pinghandler предоставляет HTTP-хендлер для проверки доступности хранилища (например, базы данных)
// через эндпоинт /pinghandler.
package pinghandler

import (
	"context"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// PingHandler обрабатывает HTTP-запросы к эндпоинту /pinghandler,
// позволяя клиентам проверить, доступно ли хранилище метрик.
type PingHandler struct {
	log     *zap.Logger
	storage service.Storage
}

// NewPingHandler создаёт новый экземпляр PingHandler.
// Принимает логгер zap.Logger и реализацию хранилища service.Storage.
// Хранилище должно дополнительно реализовывать интерфейс HealthChecker для поддержки проверки.
func NewPingHandler(log *zap.Logger, storage service.Storage) *PingHandler {
	return &PingHandler{
		log:     log,
		storage: storage,
	}
}

// GetPingDB обрабатывает GET-запрос к /pinghandler.
// Пытается выполнить health-check хранилища с таймаутом 3 секунды.
// Если хранилище не реализует HealthChecker — возвращает 500 Internal Server Error.
// Если проверка завершилась ошибкой — логирует предупреждение и возвращает 500.
// В случае успеха возвращает HTTP 200 OK без тела ответа.
func (h *PingHandler) GetPingDB(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := h.storage.Ping(ctx); err != nil {
		h.log.Warn("Failed to pinghandler database", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
