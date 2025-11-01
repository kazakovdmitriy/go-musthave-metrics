package ping

import (
	"net/http"

	"go.uber.org/zap"
)

type PingHandler struct {
	log *zap.Logger
	db  HealthChecker
}

func NewPingHandler(log *zap.Logger, db HealthChecker) *PingHandler {
	return &PingHandler{
		log: log,
		db:  db,
	}
}

func (m *PingHandler) GetPingDB(w http.ResponseWriter, r *http.Request) {
	err := m.db.Ping(r.Context())
	if err != nil {
		m.log.Warn("Failed to connect to the database", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
