package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
)

type MetricsHandler struct {
	service MetricsService
}

func NewMetricsHandler(service MetricsService) *MetricsHandler {
	return &MetricsHandler{
		service: service,
	}
}

func (h *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")

	var value interface{}
	var err error

	switch metricType {
	case model.Gauge:
		value, err = h.service.GetGauge(metricName)
	case model.Counter:
		value, err = h.service.GetCounter(metricName)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var strValue string
	switch v := value.(type) {
	case float64:
		strValue = strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		strValue = strconv.FormatInt(v, 10)
	default:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strValue))
}

func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	// Дебаг
	// fmt.Println("data received from endpoint: ", r.URL.Path)

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "value")

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
