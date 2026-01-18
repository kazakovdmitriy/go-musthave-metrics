package mainpageservice

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainPageService_GetMainPage_WithGomock(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		// Arrange
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStorage := mocks.NewMockStorage(ctrl)
		ctx := context.Background()
		expectedMetrics := "counter1 = 10\ngauge1 = 3.14"

		mockStorage.EXPECT().
			GetAllMetrics(ctx).
			Return(expectedMetrics, nil).
			Times(1)

		service := NewMainPageService(mockStorage)

		// Act
		result, err := service.GetMainPage(ctx)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, result, "<h1>Список метрик</h1>")
		assert.Contains(t, result, "counter1 = 10")
		assert.Contains(t, result, "gauge1 = 3.14")
	})

	t.Run("error from storage", func(t *testing.T) {
		// Arrange
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStorage := mocks.NewMockStorage(ctrl)
		ctx := context.Background()
		expectedErr := errors.New("storage unavailable")

		mockStorage.EXPECT().
			GetAllMetrics(ctx).
			Return("", expectedErr).
			Times(1)

		service := NewMainPageService(mockStorage)

		// Act
		result, err := service.GetMainPage(ctx)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Empty(t, result)
	})

	t.Run("verify HTML structure", func(t *testing.T) {
		// Arrange
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStorage := mocks.NewMockStorage(ctrl)
		ctx := context.Background()

		mockStorage.EXPECT().
			GetAllMetrics(ctx).
			Return("test", nil).
			Times(1)

		service := NewMainPageService(mockStorage)

		// Act
		result, err := service.GetMainPage(ctx)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, result, "<!DOCTYPE html>")
		assert.Contains(t, result, "<html lang=\"ru\">")
		assert.Contains(t, result, "<title>Доступные метрики</title>")
		assert.Contains(t, result, "<body>")
		assert.Contains(t, result, "</body>")
		assert.Contains(t, result, "</html>")
	})
}
