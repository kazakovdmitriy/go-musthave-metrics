package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

type MetricsHandler struct {
	service MetricsService
	log     *zap.Logger
}

func NewMetricsHandler(service MetricsService, log *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		service: service,
		log:     log,
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
		h.log.Error("invalid metric type",
			zap.String("metric_type", metricType),
			zap.String("metric_name", metricName))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err != nil {
		h.log.Error("error getting metric",
			zap.String("metric_type", metricType),
			zap.String("metric_name", metricName),
			zap.Error(err))
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
		h.log.Error("unexpected metric value type",
			zap.String("metric_type", metricType),
			zap.String("metric_name", metricName),
			zap.String("value_type", fmt.Sprintf("%T", v)))
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
		h.log.Error("error processing metric update",
			zap.String("metric_type", metricType),
			zap.String("metric_name", metricName),
			zap.String("metric_value", metricValue),
			zap.Int("status_code", status),
			zap.Error(err))
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
			h.log.Error("invalid JSON in request", zap.Error(err))
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		metricType := strings.ToLower(data.MType)
		metricName := data.ID

		switch metricType {
		case model.Gauge:
			if data.Value == nil {
				h.log.Error("gauge metric value is nil",
					zap.String("metric_name", metricName))
				http.Error(w, "metric value is required", http.StatusBadRequest)
				return
			}
			h.service.UpdateGauge(metricName, *data.Value)
		case model.Counter:
			if data.Delta == nil {
				h.log.Error("counter metric delta is nil",
					zap.String("metric_name", metricName))
				http.Error(w, "metric delta is required", http.StatusBadRequest)
				return
			}
			h.service.UpdateCounter(metricName, *data.Delta)
		default:
			h.log.Error("unknown metric type in JSON",
				zap.String("metric_type", data.MType),
				zap.String("metric_name", data.ID))
			http.Error(w, "unknown metric type", http.StatusBadRequest)
			return
		}
	default:
		h.log.Error("unsupported content type", zap.String("content_type", contentType))
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
			h.log.Error("invalid JSON in request", zap.Error(err))
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		switch strings.ToLower(data.MType) {
		case model.Gauge:
			value, err := h.service.GetGauge(data.ID)
			if err != nil {
				h.log.Error("error getting gauge metric",
					zap.String("metric_name", data.ID),
					zap.Error(err))
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
				h.log.Error("error getting counter metric",
					zap.String("metric_name", data.ID),
					zap.Error(err))
				http.Error(w, "Invalid metric value", http.StatusNotFound)
				return
			}
			resp = model.Metrics{
				ID:    data.ID,
				MType: data.MType,
				Delta: &value,
			}
		default:
			h.log.Error("unknown metric type in request",
				zap.String("metric_type", data.MType))
			http.Error(w, "unknown metric type", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			h.log.Error("error encoding response", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	default:
		h.log.Error("unsupported content type", zap.String("content_type", contentType))
		http.Error(w, "Unsupported content type", http.StatusUnsupportedMediaType)
		return
	}
}

func (h *MetricsHandler) processMetricUpdate(metricType, metricName, metricValue string) (int, error) {
	switch metricType {
	case model.Gauge:
		f, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			h.log.Error("invalid gauge value format",
				zap.String("metric_name", metricName),
				zap.String("metric_value", metricValue),
				zap.Error(err))
			return http.StatusBadRequest, err
		}
		h.service.UpdateGauge(metricName, f)

	case model.Counter:
		f, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			h.log.Error("invalid counter value format",
				zap.String("metric_name", metricName),
				zap.String("metric_value", metricValue),
				zap.Error(err))
			return http.StatusBadRequest, err
		}
		h.service.UpdateCounter(metricName, f)

	default:
		h.log.Error("unknown metric type in update",
			zap.String("metric_type", metricType),
			zap.String("metric_name", metricName))
		return http.StatusBadRequest, fmt.Errorf("unknown metric type")
	}

	return http.StatusOK, nil
}
