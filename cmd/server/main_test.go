package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetupHandler(t *testing.T) {
	handler := setupHandler()

	if handler == nil {
		t.Error("handler should not be nil")
	}
}

func TestUpdateEndpoint(t *testing.T) {
	// Создаем обработчик как в main
	handler := setupHandler()

	// Тестируем различные запросы
	testCases := []struct {
		method string
		path   string
		status int
	}{
		{"POST", "/update/gauge/test_metric/123.45", http.StatusOK},
		{"POST", "/update/counter/requests/1", http.StatusOK},
		{"GET", "/update/gauge/test_metric/123.45", http.StatusMethodNotAllowed},
		{"POST", "/update/unknown/test/123", http.StatusBadRequest},
		{"POST", "/update/gauge/test_metric/invalid", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tc.status {
				t.Errorf("expected status %d, got %d for %s", tc.status, w.Code, tc.path)
			}
		})
	}
}
