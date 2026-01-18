package metrics_test

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/metrics"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
)

type mockMetricsService struct{}

func (m mockMetricsService) GetGauge(_ context.Context, name string) (float64, error) {
	return 23.5, nil
}
func (m mockMetricsService) GetCounter(_ context.Context, name string) (int64, error) {
	return 42, nil
}
func (m mockMetricsService) UpdateGauge(_ context.Context, name string, value float64) error {
	return nil
}
func (m mockMetricsService) UpdateCounter(_ context.Context, name string, delta int64) error {
	return nil
}
func (m mockMetricsService) UpdateMetrics(_ context.Context, metrics []model.Metrics, remoteAddr string) error {
	return nil
}

func ExampleMetricsHandler_GetMetric_gauge() {
	log, _ := zap.NewDevelopment()
	handler := metrics.NewMetricsHandler(mockMetricsService{}, log)

	r := chi.NewRouter()
	r.Get("/value/{metricType}/{metricName}", handler.GetMetric)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Temperature", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Output:
	// 200
	// 23.5
	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
}
