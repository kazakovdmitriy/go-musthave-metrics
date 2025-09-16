package service

import (
	"fmt"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/repository"
)

type mainPageService struct {
	storage repository.Storage
}

func NewMainPageService(storage repository.Storage) *mainPageService {
	return &mainPageService{
		storage: storage,
	}
}

func (s *mainPageService) GetMainPage() (string, error) {
	metricsResult, err := s.storage.GetAllMetrics()
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
