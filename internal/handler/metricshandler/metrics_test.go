package metricshandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/mocks"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"

	"go.uber.org/zap"
)

func TestMetricsHandler_GetMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockMetricsService(ctrl)
	logger := zap.NewNop()
	handler := NewMetricsHandler(mockService, logger)

	tests := []struct {
		name           string
		metricType     string
		metricName     string
		setupMock      func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name:       "success gauge",
			metricType: "gauge",
			metricName: "test_gauge",
			setupMock: func() {
				mockService.EXPECT().GetGauge(gomock.Any(), "test_gauge").Return(123.45, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "123.45",
		},
		{
			name:       "success counter",
			metricType: "counter",
			metricName: "test_counter",
			setupMock: func() {
				mockService.EXPECT().GetCounter(gomock.Any(), "test_counter").Return(int64(42), nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "42",
		},
		{
			name:           "invalid metric type",
			metricType:     "invalid",
			metricName:     "test",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid metric type\n",
		},
		{
			name:       "gauge not found",
			metricType: "gauge",
			metricName: "nonexistent",
			setupMock: func() {
				mockService.EXPECT().GetGauge(gomock.Any(), "nonexistent").Return(0.0, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "error getting gauge metric\n",
		},
		{
			name:       "counter not found",
			metricType: "counter",
			metricName: "nonexistent",
			setupMock: func() {
				mockService.EXPECT().GetCounter(gomock.Any(), "nonexistent").Return(int64(0), errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "error getting counter metric\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			r := chi.NewRouter()
			r.Get("/value/{metricType}/{metricName}", handler.GetMetric)

			req := httptest.NewRequest("GET", fmt.Sprintf("/value/%s/%s", tt.metricType, tt.metricName), nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestMetricsHandler_UpdateMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockMetricsService(ctrl)
	logger := zap.NewNop()
	handler := NewMetricsHandler(mockService, logger)

	tests := []struct {
		name           string
		metricType     string
		metricName     string
		value          string
		setupMock      func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name:       "success gauge",
			metricType: "gauge",
			metricName: "test_gauge",
			value:      "123.45",
			setupMock: func() {
				mockService.EXPECT().UpdateGauge(gomock.Any(), "test_gauge", 123.45).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "success counter",
			metricType: "counter",
			metricName: "test_counter",
			value:      "42",
			setupMock: func() {
				mockService.EXPECT().UpdateCounter(gomock.Any(), "test_counter", int64(42)).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid metric type",
			metricType:     "invalid",
			metricName:     "test",
			value:          "123",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "unknown metric type\n",
		},
		{
			name:           "invalid gauge value",
			metricType:     "gauge",
			metricName:     "test_gauge",
			value:          "invalid",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid gauge value format\n",
		},
		{
			name:           "invalid counter value",
			metricType:     "counter",
			metricName:     "test_counter",
			value:          "invalid",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid counter value format\n",
		},
		{
			name:       "gauge update error",
			metricType: "gauge",
			metricName: "test_gauge",
			value:      "123.45",
			setupMock: func() {
				mockService.EXPECT().UpdateGauge(gomock.Any(), "test_gauge", 123.45).Return(errors.New("update failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to update gauge metric\n",
		},
		{
			name:       "counter update error",
			metricType: "counter",
			metricName: "test_counter",
			value:      "42",
			setupMock: func() {
				mockService.EXPECT().UpdateCounter(gomock.Any(), "test_counter", int64(42)).Return(errors.New("update failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to update counter metric\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			r := chi.NewRouter()
			r.Post("/update/{metricType}/{metricName}/{value}", handler.UpdateMetric)

			req := httptest.NewRequest("POST", fmt.Sprintf("/update/%s/%s/%s", tt.metricType, tt.metricName, tt.value), nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestMetricsHandler_UpdatePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockMetricsService(ctrl)
	logger := zap.NewNop()
	handler := NewMetricsHandler(mockService, logger)

	tests := []struct {
		name           string
		contentType    string
		body           interface{}
		setupMock      func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "success gauge",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test_gauge",
				MType: "gauge",
				Value: float64Ptr(123.45),
			},
			setupMock: func() {
				mockService.EXPECT().UpdateGauge(gomock.Any(), "test_gauge", 123.45).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "success counter",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test_counter",
				MType: "counter",
				Delta: int64Ptr(42),
			},
			setupMock: func() {
				mockService.EXPECT().UpdateCounter(gomock.Any(), "test_counter", int64(42)).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unsupported content type",
			contentType:    "text/plain",
			body:           nil,
			setupMock:      func() {},
			expectedStatus: http.StatusUnsupportedMediaType,
			expectedBody:   "unsupported content type\n",
		},
		{
			name:           "invalid JSON",
			contentType:    "application/json",
			body:           "invalid json",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid JSON in request\n",
		},
		{
			name:        "unknown metric type",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test",
				MType: "invalid",
			},
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "unknown metric type\n",
		},
		{
			name:        "gauge value is nil",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test_gauge",
				MType: "gauge",
				Value: nil,
			},
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "metric value is required for gauge\n",
		},
		{
			name:        "counter delta is nil",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test_counter",
				MType: "counter",
				Delta: nil,
			},
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "metric delta is required for counter\n",
		},
		{
			name:        "gauge update error",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test_gauge",
				MType: "gauge",
				Value: float64Ptr(123.45),
			},
			setupMock: func() {
				mockService.EXPECT().UpdateGauge(gomock.Any(), "test_gauge", 123.45).Return(errors.New("update failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to update gauge metric\n",
		},
		{
			name:        "counter update error",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test_counter",
				MType: "counter",
				Delta: int64Ptr(42),
			},
			setupMock: func() {
				mockService.EXPECT().UpdateCounter(gomock.Any(), "test_counter", int64(42)).Return(errors.New("update failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to update counter metric\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var bodyBytes []byte
			switch b := tt.body.(type) {
			case string:
				bodyBytes = []byte(b)
			default:
				var err error
				bodyBytes, err = json.Marshal(b)
				if err != nil {
					t.Fatalf("failed to marshal body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/update/", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			handler.UpdatePost(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestMetricsHandler_SentMetricPost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockMetricsService(ctrl)
	logger := zap.NewNop()
	handler := NewMetricsHandler(mockService, logger)

	tests := []struct {
		name           string
		contentType    string
		body           interface{}
		setupMock      func()
		expectedStatus int
		expectedBody   string
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:        "success gauge",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test_gauge",
				MType: "gauge",
			},
			setupMock: func() {
				mockService.EXPECT().GetGauge(gomock.Any(), "test_gauge").Return(123.45, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp model.Metrics
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if resp.Value == nil || *resp.Value != 123.45 {
					t.Errorf("expected value 123.45, got %v", resp.Value)
				}
			},
		},
		{
			name:        "success counter",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test_counter",
				MType: "counter",
			},
			setupMock: func() {
				mockService.EXPECT().GetCounter(gomock.Any(), "test_counter").Return(int64(42), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp model.Metrics
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if resp.Delta == nil || *resp.Delta != 42 {
					t.Errorf("expected delta 42, got %v", resp.Delta)
				}
			},
		},
		{
			name:           "unsupported content type",
			contentType:    "text/plain",
			body:           nil,
			setupMock:      func() {},
			expectedStatus: http.StatusUnsupportedMediaType,
			expectedBody:   "unsupported content type\n",
		},
		{
			name:           "invalid JSON",
			contentType:    "application/json",
			body:           "invalid json",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid JSON in request\n",
		},
		{
			name:        "unknown metric type",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test",
				MType: "invalid",
			},
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "unknown metric type\n",
		},
		{
			name:        "gauge not found",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test_gauge",
				MType: "gauge",
			},
			setupMock: func() {
				mockService.EXPECT().GetGauge(gomock.Any(), "test_gauge").Return(0.0, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "metric not found\n",
		},
		{
			name:        "counter not found",
			contentType: "application/json",
			body: model.Metrics{
				ID:    "test_counter",
				MType: "counter",
			},
			setupMock: func() {
				mockService.EXPECT().GetCounter(gomock.Any(), "test_counter").Return(int64(0), errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "metric not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var bodyBytes []byte
			switch b := tt.body.(type) {
			case string:
				bodyBytes = []byte(b)
			default:
				var err error
				bodyBytes, err = json.Marshal(b)
				if err != nil {
					t.Fatalf("failed to marshal body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/value/", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			handler.SentMetricPost(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, w.Body.String())
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String())
			}
		})
	}
}

func Test_isValidMetricType(t *testing.T) {
	tests := []struct {
		name       string
		metricType string
		expected   bool
	}{
		{"gauge valid", "gauge", true},
		{"counter valid", "counter", true},
		{"invalid type", "invalid", false},
		{"empty type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidMetricType(tt.metricType)
			if result != tt.expected {
				t.Errorf("isValidMetricType(%q) = %v, expected %v", tt.metricType, result, tt.expected)
			}
		})
	}
}

// Вспомогательные функции
func float64Ptr(f float64) *float64 {
	return &f
}

func int64Ptr(i int64) *int64 {
	return &i
}
