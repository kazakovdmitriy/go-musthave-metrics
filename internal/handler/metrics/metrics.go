// Package metrics предоставляет HTTP-хендлеры для работы с метриками приложения:
// получение, обновление и пакетное сохранение значений типа gauge и counter.
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

// MetricsHandler обрабатывает HTTP-запросы, связанные с получением и обновлением метрик.
// Поддерживает как URL-параметры, так и JSON-тело запроса.
type MetricsHandler struct {
	service MetricsService
	log     *zap.Logger
}

// NewMetricsHandler создаёт новый экземпляр MetricsHandler.
// Принимает реализацию MetricsService и логгер zap.Logger.
// Не проверяет service на nil — ожидается, что он всегда передаётся корректно.
func NewMetricsHandler(service MetricsService, log *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		service: service,
		log:     log,
	}
}

// GetMetric обрабатывает GET-запрос вида /value/{metricType}/{metricName}.
// Возвращает текстовое представление значения метрики (gauge или counter).
// Устанавливает Content-Type: text/plain.
// В случае ошибки (неверный тип метрики, метрика не найдена) логирует событие и возвращает соответствующий HTTP-статус.
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

// UpdateMetric обрабатывает PUT-запрос вида /update/{metricType}/{metricName}/{value}.
// Обновляет значение метрики на основе переданного строкового значения.
// Поддерживает только gauge (float64) и counter (int64).
// При ошибке парсинга или сохранения логирует событие и возвращает соответствующий HTTP-статус.
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

// UpdatePost обрабатывает POST-запрос к эндпоинту /update.
// Ожидает тело запроса в формате JSON, соответствующее структуре model.Metrics.
// Обновляет одну метрику (gauge или counter) в зависимости от переданного MType.
// Требует Content-Type: application/json.
// В случае ошибки декодирования JSON, отсутствия значения или ошибки сохранения — логирует и возвращает ошибку.
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

// UpdateMetrics обрабатывает POST-запрос к эндпоинту /updates.
// Принимает массив метрик в формате JSON и выполняет их пакетное обновление через сервис.
// Требует Content-Type: application/json.
// В случае ошибки декодирования или сохранения — логирует и возвращает ошибку.
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

// SentMetricPost обрабатывает POST-запрос к эндпоинту /value.
// Принимает описание метрики в формате JSON и возвращает её текущее значение в том же формате.
// Используется для получения актуального состояния метрики после её возможного обновления.
// Требует Content-Type: application/json.
// В случае ошибки — логирует и возвращает соответствующий HTTP-статус.
// Ответ сериализуется в JSON с Content-Type: application/json.
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

// isValidMetricType проверяет, является ли переданная строка допустимым типом метрики.
// Допустимые значения: model.Gauge ("gauge") и model.Counter ("counter").
func isValidMetricType(metricType string) bool {
	return metricType == model.Gauge || metricType == model.Counter
}

// updateMetricByType обновляет метрику по типу, имени и строковому значению.
// Выполняет парсинг значения и вызывает соответствующий метод сервиса.
// При ошибках логирует и отправляет HTTP-ответ клиенту.
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

// updateMetricFromJSON обновляет одну метрику на основе структуры model.Metrics.
// Проверяет наличие обязательных полей (Value для gauge, Delta для counter).
// При ошибках логирует и отправляет HTTP-ответ клиенту.
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

// getMetricForJSONResponse извлекает текущее значение метрики по данным из запроса
// и формирует ответ в виде model.Metrics для последующей сериализации в JSON.
// При ошибках логирует и отправляет HTTP-ответ клиенту.
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

// logAndWriteError логирует ошибку с дополнительными полями и отправляет HTTP-ошибку клиенту.
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
