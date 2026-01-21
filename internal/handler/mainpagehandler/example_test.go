// internal/handler/mainpagehandler/example_test.go
package mainpagehandler_test

import (
	"context"
	"fmt"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/mainpagehandler"
	"net/http"
	"net/http/httptest"
)

type mockMainPageService struct{}

func (m mockMainPageService) GetMainPage(_ context.Context) (string, error) {
	return "<h1>Hello</h1>", nil
}

func ExampleMainPageHandler_GetMainPage() {
	handler := mainpagehandler.NewMainPageHandler(mockMainPageService{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.GetMainPage(w, req)

	// Output:
	// 200
	// <h1>Hello</h1>
	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
}
