// example_test.go
package pinghandler_test

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/mocks"
	"net/http"
	"net/http/httptest"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/pinghandler"
	"go.uber.org/zap"
)

// ExamplePingHandler_GetPingDB demonstrates how to use the /pinghandler endpoint.
func ExamplePingHandler_GetPingDB() {
	ctrl := gomock.NewController(nil)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockStorage.EXPECT().Ping(gomock.Any()).Return(nil)

	log, _ := zap.NewDevelopment()
	handler := pinghandler.NewPingHandler(log, mockStorage)

	req := httptest.NewRequest(http.MethodGet, "/pinghandler/", nil)
	w := httptest.NewRecorder()

	handler.GetPingDB(w, req)

	// Output: 200
	fmt.Println(w.Code)
}
