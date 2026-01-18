package mainpageservice

import (
	"context"
	"html/template"
	"strings"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
)

type mainPageService struct {
	storage  service.Storage
	template *template.Template
}

func NewMainPageService(storage service.Storage) (*mainPageService, error) {
	tmpl, err := template.New("mainpage").Parse(`<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Доступные метрики</title>
</head>
<body>
    <h1>Список метрик</h1>
    <p>{{.}}</p>
</body>
</html>`)

	if err != nil {
		return nil, err
	}

	return &mainPageService{
		storage:  storage,
		template: tmpl,
	}, nil
}

func (s *mainPageService) GetMainPage(ctx context.Context) (string, error) {
	metricsResult, err := s.storage.GetAllMetrics(ctx)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	if err := s.template.Execute(&builder, metricsResult); err != nil {
		return "", err
	}

	return builder.String(), nil
}
