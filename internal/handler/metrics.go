package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
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
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "value")

	if status, err := h.processMetricUpdate(metricType, metricName, metricValue); err != nil {
		if status == http.StatusBadRequest {
			w.WriteHeader(status)
			return
		}
		w.WriteHeader(status)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {

	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "application/json"):
		var data model.Metrics
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		metricType := strings.ToLower(data.MType)
		metricName := data.ID

		switch metricType {
		case model.Gauge:
			h.service.UpdateGauge(metricName, *data.Value)
		case model.Counter:
			h.service.UpdateCounter(metricName, *data.Delta)
		default:
			http.Error(w, "unknown metric type", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "Unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) SentMetricPost(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "application/json"):

		var data model.Metrics
		var resp model.Metrics

		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid JSON", http.StatusNotFound)
			return
		}

		switch strings.ToLower(data.MType) {
		case model.Gauge:
			value, err := h.service.GetGauge(data.ID)
			if err != nil {
				http.Error(w, "Invalid metric value", http.StatusNotFound)
				return
			}

			resp = model.Metrics{
				ID:    data.ID,
				MType: data.MType,
				Value: &value,
			}
		case strings.ToLower(model.Counter):
			value, err := h.service.GetCounter(data.ID)
			if err != nil {
				http.Error(w, "Invalid metric value", http.StatusNotFound)
				return
			}
			resp = model.Metrics{
				ID:    data.ID,
				MType: data.MType,
				Delta: &value,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			logger.Log.Error("error encoding response", zap.Error(err))
			return
		}
	}
}

func (h *MetricsHandler) processMetricUpdate(metricType, metricName, metricValue string) (int, error) {
	switch metricType {
	case model.Gauge:
		f, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return http.StatusBadRequest, err
		}
		h.service.UpdateGauge(metricName, f)

	case model.Counter:
		f, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return http.StatusBadRequest, err
		}
		h.service.UpdateCounter(metricName, f)

	default:
		return http.StatusBadRequest, fmt.Errorf("unknown metric type")
	}

	return http.StatusOK, nil
}

// func writeErrorBadRequests(w http.ResponseWriter, err error) bool {
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return true
// 	}
// 	return false
// }
