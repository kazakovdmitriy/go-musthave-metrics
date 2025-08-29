package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
)

type MetricsHandler struct {
	service service.MetricsServer
}

func NewMetricsHandler(service service.MetricsServer) *MetricsHandler {
	return &MetricsHandler{
		service: service,
	}
}

func (h *MetricsHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(pathParts) < 4 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	metricType := pathParts[1]
	metricName := pathParts[2]
	metricValue := pathParts[3]

	switch metricType {
	case model.Gauge:

		f, err := strconv.ParseFloat(metricValue, 64)

		if writeErrorBadRequests(w, err) {
			return
		}

		h.service.UpdateGauge(metricName, f)
	case model.Counter:
		f, err := strconv.ParseInt(metricValue, 10, 64)

		if writeErrorBadRequests(w, err) {
			return
		}

		h.service.UpdateCounter(metricName, f)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Для дебага
	// fmt.Println(r.URL.Path)

	w.WriteHeader(http.StatusOK)
}

func writeErrorBadRequests(w http.ResponseWriter, err error) bool {
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return true
	}
	return false
}
