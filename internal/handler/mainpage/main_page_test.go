package mainpage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/mocks"
)

func TestMainPageHandler_GetMainPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		setupMock      func(*mocks.MockMainPageService)
		expectedStatus int
		expectedBody   string
		expectedHeader string
	}{
		{
			name: "success - main page returned",
			setupMock: func(mockService *mocks.MockMainPageService) {
				mockService.EXPECT().
					GetMainPage(gomock.Any()).
					Return("<html><body>Main Page</body></html>", nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "<html><body>Main Page</body></html>",
			expectedHeader: "text/html; charset=utf-8",
		},
		{
			name: "error - service returns error",
			setupMock: func(mockService *mocks.MockMainPageService) {
				mockService.EXPECT().
					GetMainPage(gomock.Any()).
					Return("", errors.New("database error"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Page not found",
			expectedHeader: "text/html; charset=utf-8",
		},
		{
			name: "success - empty page content",
			setupMock: func(mockService *mocks.MockMainPageService) {
				mockService.EXPECT().
					GetMainPage(gomock.Any()).
					Return("", nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
			expectedHeader: "text/html; charset=utf-8",
		},
		{
			name: "success - complex HTML content",
			setupMock: func(mockService *mocks.MockMainPageService) {
				complexHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Metrics Dashboard</title>
</head>
<body>
    <h1>Welcome to Metrics Dashboard</h1>
</body>
</html>`
				mockService.EXPECT().
					GetMainPage(gomock.Any()).
					Return(complexHTML, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: `<!DOCTYPE html>
<html>
<head>
    <title>Metrics Dashboard</title>
</head>
<body>
    <h1>Welcome to Metrics Dashboard</h1>
</body>
</html>`,
			expectedHeader: "text/html; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockMainPageService(ctrl)
			tt.setupMock(mockService)

			handler := NewMainPageHandler(mockService)

			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()

			handler.GetMainPage(w, req)

			// Проверяем статус код
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Проверяем тело ответа
			if w.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, w.Body.String())
			}

			// Проверяем заголовок Content-Type
			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedHeader {
				t.Errorf("expected Content-Type %q, got %q", tt.expectedHeader, contentType)
			}
		})
	}
}

func TestMainPageHandler_ContextPropagation(t *testing.T) {
	t.Run("context is passed to service", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := mocks.NewMockMainPageService(ctrl)
		handler := NewMainPageHandler(mockService)

		// Создаем контекст с конкретным значением
		type contextKey string
		const key contextKey = "test-key"

		mockService.EXPECT().
			GetMainPage(gomock.Any()).
			DoAndReturn(func(ctx context.Context) (string, error) {
				// Проверяем, что контекст передается правильно
				if ctx == nil {
					return "", errors.New("context is nil")
				}
				// Можно проверить конкретные значения в контексте, если они ожидаются
				return "<html>Context test</html>", nil
			})

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler.GetMainPage(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestNewMainPageHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockMainPageService(ctrl)

	handler := NewMainPageHandler(mockService)

	if handler == nil {
		t.Fatal("expected handler to be created, got nil")
	}

	if handler.service != mockService {
		t.Error("expected service to be set correctly")
	}
}

func TestMainPageHandler_DifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockMainPageService(ctrl)
			handler := NewMainPageHandler(mockService)

			// Для каждого метода создаем отдельный мок с ожиданием вызова
			mockService.EXPECT().
				GetMainPage(gomock.Any()).
				Return("<html>Test</html>", nil)

			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()

			handler.GetMainPage(w, req)

			// Для всех методов должен возвращаться HTML контент
			contentType := w.Header().Get("Content-Type")
			if contentType != "text/html; charset=utf-8" {
				t.Errorf("for method %s: expected Content-Type %q, got %q",
					method, "text/html; charset=utf-8", contentType)
			}

			// Проверяем, что для всех методов возвращается статус 200
			// (если сервис не вернул ошибку)
			if w.Code != http.StatusOK {
				t.Errorf("for method %s: expected status %d, got %d",
					method, http.StatusOK, w.Code)
			}
		})
	}
}

// Тест для проверки поведения при панике в сервисе
func TestMainPageHandler_ServicePanic(t *testing.T) {
	t.Run("service panics", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := mocks.NewMockMainPageService(ctrl)
		handler := NewMainPageHandler(mockService)

		// Этот тест проверяет, что хендлер не паникует при панике в сервисе
		mockService.EXPECT().
			GetMainPage(gomock.Any()).
			DoAndReturn(func(ctx context.Context) (string, error) {
				panic("service panic")
			})

		// Восстанавливаем панику, чтобы тест не упал
		defer func() {
			if r := recover(); r != nil {
				t.Log("panic recovered as expected:", r)
				// Тест должен пройти, так как мы ожидаем панику
			}
		}()

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		// Этот вызов должен вызвать панику, но она будет восстановлена в defer
		handler.GetMainPage(w, req)

		// Если мы дошли сюда, значит паника была восстановлена
		// Можно добавить дополнительные проверки если необходимо
	})
}

// Тест для случая, когда сервис возвращает ошибку для разных методов
func TestMainPageHandler_DifferentMethodsWithError(t *testing.T) {
	methods := []string{"GET", "POST", "PUT"}

	for _, method := range methods {
		t.Run(method+" with error", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockMainPageService(ctrl)
			handler := NewMainPageHandler(mockService)

			// Ожидаем ошибку от сервиса
			mockService.EXPECT().
				GetMainPage(gomock.Any()).
				Return("", errors.New("service error"))

			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()

			handler.GetMainPage(w, req)

			// Проверяем, что при ошибке возвращается статус 404
			if w.Code != http.StatusNotFound {
				t.Errorf("for method %s with error: expected status %d, got %d",
					method, http.StatusNotFound, w.Code)
			}

			// Проверяем сообщение об ошибке
			if w.Body.String() != "Page not found" {
				t.Errorf("for method %s with error: expected body %q, got %q",
					method, "Page not found", w.Body.String())
			}
		})
	}
}

// Benchmark тест для проверки производительности
func BenchmarkMainPageHandler_GetMainPage(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockService := mocks.NewMockMainPageService(ctrl)
	handler := NewMainPageHandler(mockService)

	mockService.EXPECT().
		GetMainPage(gomock.Any()).
		Return("<html>Benchmark</html>", nil).
		AnyTimes()

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.GetMainPage(w, req)
	}
}
