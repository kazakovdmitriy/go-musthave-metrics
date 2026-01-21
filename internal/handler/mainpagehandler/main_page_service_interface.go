package mainpagehandler

import "context"

// MainPageService определяет контракт для получения содержимого главной страницы.
// Должен быть реализован внешним сервисом, интегрируемым с MainPageHandler.
type MainPageService interface {
	// GetMainPage возвращает HTML-содержимое главной страницы.
	// Принимает контекст запроса для поддержки отмены и таймаутов.
	// Возвращает строку с HTML и ошибку, если страница недоступна.
	GetMainPage(ctx context.Context) (string, error)
}
