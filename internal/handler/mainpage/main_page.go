// Package mainpage предоставляет HTTP-хендлер для отображения главной страницы.
package mainpage

import (
	"net/http"
)

// MainPageHandler обрабатывает HTTP-запросы, связанные с отображением главной страницы.
type MainPageHandler struct {
	service MainPageService
}

// NewMainPageHandler создаёт новый экземпляр MainPageHandler.
// Принимает реализацию MainPageService. Если service равен nil,
// функция возвращает nil, предотвращая дальнейшее использование неинициализированного хендлера.
func NewMainPageHandler(service MainPageService) *MainPageHandler {

	if service == nil {
		return nil
	}
	return &MainPageHandler{service: service}
}

// GetMainPage обрабатывает GET-запрос к главной странице.
// Извлекает HTML-контент через внедрённый MainPageService и отправляет его клиенту.
// Устанавливает заголовок Content-Type: text/html; charset=utf-8.
// В случае ошибки при получении страницы возвращает HTTP 404 с сообщением "Page not found".
// В случае успеха возвращает HTTP 200 и HTML-контент.
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
