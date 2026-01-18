package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
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

	if !isValidMetricType(metricType) {
		h.logAndWriteError(
			w,
			fmt.Errorf("invalid metric type: %s", metricType),
			http.StatusBadRequest,
			"invalid metric type",
			zap.String("metric_type", metricType),
			zap.String("metric_name", metricName),
		)
		return
	}

	var valueStr string

	switch metricType {
	case model.Gauge:
		gaugeValue, err := h.service.GetGauge(r.Context(), metricName)
		if err != nil {
			h.logAndWriteError(w, err, http.StatusNotFound, "error getting gauge metric",
				zap.String("metric_type", metricType), zap.String("metric_name", metricName))
			return
		}
		valueStr = strconv.FormatFloat(gaugeValue, 'f', -1, 64)

	case model.Counter:
		counterValue, err := h.service.GetCounter(r.Context(), metricName)
		if err != nil {
			h.logAndWriteError(w, err, http.StatusNotFound, "error getting counter metric",
				zap.String("metric_type", metricType), zap.String("metric_name", metricName))
			return
		}
		valueStr = strconv.FormatInt(counterValue, 10)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(valueStr))
}

func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValueStr := chi.URLParam(r, "value")

	err := h.updateMetricByType(w, r.Context(), metricType, metricName, metricValueStr)
	if err != nil {
		// updateMetricByType уже логирует ошибки и вызывает http.Error.
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		h.logAndWriteError(
			w,
			fmt.Errorf("unsupported content type"),
			http.StatusUnsupportedMediaType,
			"unsupported content type",
			zap.String("content_type", r.Header.Get("Content-Type")),
		)
		return
	}

	if r.Body == nil || r.ContentLength == 0 {
		h.logAndWriteError(
			w,
			fmt.Errorf("empty request body"),
			http.StatusBadRequest,
			"empty request body",
		)
		return
	}

	var data model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.logAndWriteError(w, err, http.StatusBadRequest, "invalid JSON in request", zap.Error(err))
		return
	}

	if err := h.updateMetricFromJSON(w, r.Context(), data); err != nil {
		// updateMetricFromJSON уже логирует ошибки и вызывает http.Error.
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		h.logAndWriteError(
			w,
			fmt.Errorf("unsupported content type"),
			http.StatusUnsupportedMediaType,
			"unsupported content type",
			zap.String("content_type", r.Header.Get("Content-Type")),
		)
		return
	}

	var data []model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.logAndWriteError(w, err, http.StatusBadRequest, "invalid JSON in request", zap.Error(err))
		return
	}

	if err := h.service.UpdateMetrics(r.Context(), data, r.RemoteAddr); err != nil {
		h.logAndWriteError(w, err, http.StatusInternalServerError, "failed to save batch of metrics", zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) SentMetricPost(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		h.logAndWriteError(
			w,
			fmt.Errorf("unsupported content type"),
			http.StatusUnsupportedMediaType,
			"unsupported content type",
			zap.String("content_type", r.Header.Get("Content-Type")),
		)
		return
	}

	var data model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.logAndWriteError(w, err, http.StatusBadRequest, "invalid JSON in request", zap.Error(err))
		return
	}

	resp, err := h.getMetricForJSONResponse(w, r.Context(), data)
	if err != nil {
		// getMetricForJSONResponse уже логирует ошибки и вызывает http.Error.
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logAndWriteError(w, err, http.StatusInternalServerError, "error encoding response", zap.Error(err))
	}
}

// --- Helper functions ---

func isValidMetricType(metricType string) bool {
	return metricType == model.Gauge || metricType == model.Counter
}

func (h *MetricsHandler) updateMetricByType(
	w http.ResponseWriter,
	ctx context.Context,
	metricType,
	metricName,
	metricValueStr string,
) error {
	var err error
	switch metricType {
	case model.Gauge:
		parsedValue, parseErr := strconv.ParseFloat(metricValueStr, 64)
		if parseErr != nil {
			err = parseErr
			h.log.Error("invalid gauge value format",
				zap.String("metric_name", metricName),
				zap.String("metric_value", metricValueStr),
				zap.Error(err))
			http.Error(w, "invalid gauge value format", http.StatusBadRequest)
			return err
		}
		err = h.service.UpdateGauge(ctx, metricName, parsedValue)
		if err != nil {
			h.log.Error("error updating gauge metric",
				zap.String("metric_name", metricName),
				zap.Float64("metric_value", parsedValue),
				zap.Error(err))
			http.Error(w, "failed to update gauge metric", http.StatusInternalServerError)
			return err
		}

	case model.Counter:
		parsedValue, parseErr := strconv.ParseInt(metricValueStr, 10, 64)
		if parseErr != nil {
			err = parseErr
			h.log.Error("invalid counter value format",
				zap.String("metric_name", metricName),
				zap.String("metric_value", metricValueStr),
				zap.Error(err))
			http.Error(w, "invalid counter value format", http.StatusBadRequest)
			return err
		}
		err = h.service.UpdateCounter(ctx, metricName, parsedValue)
		if err != nil {
			h.log.Error("error updating counter metric",
				zap.String("metric_name", metricName),
				zap.Int64("metric_value", parsedValue),
				zap.Error(err))
			http.Error(w, "failed to update counter metric", http.StatusInternalServerError)
			return err
		}

	default:
		err = fmt.Errorf("unknown metric type: %s", metricType)
		h.log.Error("unknown metric type in update",
			zap.String("metric_type", metricType),
			zap.String("metric_name", metricName))
		http.Error(w, "unknown metric type", http.StatusBadRequest)
		return err
	}
	return nil
}

func (h *MetricsHandler) updateMetricFromJSON(
	w http.ResponseWriter,
	ctx context.Context,
	data model.Metrics,
) error {
	metricType := strings.ToLower(data.MType)
	metricName := data.ID

	if !isValidMetricType(metricType) {
		err := fmt.Errorf("unknown metric type: %s", data.MType)
		h.log.Error("unknown metric type in JSON",
			zap.String("metric_type", data.MType),
			zap.String("metric_name", data.ID))
		http.Error(w, "unknown metric type", http.StatusBadRequest)
		return err
	}

	switch metricType {
	case model.Gauge:
		if data.Value == nil {
			err := fmt.Errorf("gauge metric value is nil")
			h.log.Error("gauge metric value is nil",
				zap.String("metric_name", metricName))
			http.Error(w, "metric value is required for gauge", http.StatusBadRequest)
			return err
		}
		err := h.service.UpdateGauge(ctx, metricName, *data.Value)
		if err != nil {
			h.log.Error("error updating gauge metric from JSON",
				zap.String("metric_name", metricName),
				zap.Float64("metric_value", *data.Value),
				zap.Error(err))
			http.Error(w, "failed to update gauge metric", http.StatusInternalServerError)
			return err
		}

	case model.Counter:
		if data.Delta == nil {
			err := fmt.Errorf("counter metric delta is nil")
			h.log.Error("counter metric delta is nil",
				zap.String("metric_name", metricName))
			http.Error(w, "metric delta is required for counter", http.StatusBadRequest)
			return err
		}
		err := h.service.UpdateCounter(ctx, metricName, *data.Delta)
		if err != nil {
			h.log.Error("error updating counter metric from JSON",
				zap.String("metric_name", metricName),
				zap.Int64("metric_delta", *data.Delta),
				zap.Error(err))
			http.Error(w, "failed to update counter metric", http.StatusInternalServerError)
			return err
		}
	}
	return nil
}

func (h *MetricsHandler) getMetricForJSONResponse(
	w http.ResponseWriter,
	ctx context.Context,
	data model.Metrics,
) (model.Metrics, error) {
	var resp model.Metrics
	var err error
	metricType := strings.ToLower(data.MType)

	if !isValidMetricType(metricType) {
		err = fmt.Errorf("unknown metric type: %s", data.MType)
		h.log.Error("unknown metric type in request",
			zap.String("metric_type", data.MType))
		http.Error(w, "unknown metric type", http.StatusBadRequest)
		return resp, err
	}

	switch metricType {
	case model.Gauge:
		gaugeValue, getErr := h.service.GetGauge(ctx, data.ID)
		if getErr != nil {
			err = getErr
			h.log.Error("error getting gauge metric",
				zap.String("metric_name", data.ID),
				zap.Error(err))
			http.Error(w, "metric not found", http.StatusNotFound)
			return resp, err
		}
		resp = model.Metrics{
			ID:    data.ID,
			MType: data.MType,
			Value: &gaugeValue,
		}

	case model.Counter:
		counterValue, getErr := h.service.GetCounter(ctx, data.ID)
		if getErr != nil {
			err = getErr
			h.log.Error("error getting counter metric",
				zap.String("metric_name", data.ID),
				zap.Error(err))
			http.Error(w, "metric not found", http.StatusNotFound)
			return resp, err
		}
		resp = model.Metrics{
			ID:    data.ID,
			MType: data.MType,
			Delta: &counterValue,
		}
	}

	return resp, nil
}

func (h *MetricsHandler) logAndWriteError(
	w http.ResponseWriter,
	err error,
	statusCode int,
	msg string,
	fields ...zap.Field,
) {
	logEntry := h.log.With(fields...)
	logEntry.Error(msg, zap.Error(err))
	http.Error(w, msg, statusCode)
}
