package handler

import (
	"net/http"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
)

type MainPageHandler struct {
	service service.MainPageService
}

func NewMainPageHandler(service service.MainPageService) *MainPageHandler {
	return &MainPageHandler{
		service: service,
	}
}

func (h *MainPageHandler) GetMainPage(w http.ResponseWriter, r *http.Request) {
	mainPage, err := h.service.GetMainPage()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Page not found"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(mainPage))
}
