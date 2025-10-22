package mainpageservice

import (
	"context"
	"fmt"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
)

type mainPageService struct {
	storage service.Storage
}

func NewMainPageService(storage service.Storage) *mainPageService {
	return &mainPageService{
		storage: storage,
	}
}

func (s *mainPageService) GetMainPage(ctx context.Context) (string, error) {
	metricsResult, err := s.storage.GetAllMetrics(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Доступные метрики</title>
</head>
<body>
    <h1>Список метрик</h1>
    <p>%s</p>
</body>
</html>
`, metricsResult), nil
}
