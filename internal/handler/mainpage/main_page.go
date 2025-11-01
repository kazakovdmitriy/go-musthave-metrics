package mainpage

import (
	"net/http"
)

type MainPageHandler struct {
	service MainPageService
}

func NewMainPageHandler(service MainPageService) *MainPageHandler {

	if service == nil {
		return nil
	}
	return &MainPageHandler{service: service}
}

func (h *MainPageHandler) GetMainPage(w http.ResponseWriter, r *http.Request) {
	mainPage, err := h.service.GetMainPage(r.Context())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Page not found"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(mainPage))
}
