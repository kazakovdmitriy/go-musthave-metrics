// internal/handler/mainpage/example_test.go
package mainpage_test

import (
	"context"
	"fmt"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/mainpage"
	"net/http"
	"net/http/httptest"
)

type mockMainPageService struct{}

func (m mockMainPageService) GetMainPage(_ context.Context) (string, error) {
	return "<h1>Hello</h1>", nil
}

func ExampleMainPageHandler_GetMainPage() {
	handler := mainpage.NewMainPageHandler(mockMainPageService{})
	if handler == nil {
		panic("handler is nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.GetMainPage(w, req)

	// Output:
	// 200
	// <h1>Hello</h1>
	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
}
