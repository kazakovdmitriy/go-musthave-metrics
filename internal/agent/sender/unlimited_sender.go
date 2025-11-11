// sender/unlimited_sender.go
package sender

import (
	"context"
	"sync"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

// unlimitedSender отправляет метрики без ограничений (без worker pool)
type unlimitedSender struct {
	metricsService *metricsService
	logger         *zap.Logger
	wg             sync.WaitGroup
}

// newUnlimitedSender создает неограниченный отправитель
func newUnlimitedSender(client interfaces.HTTPClient, logger *zap.Logger) interfaces.MetricsSender {
	return &unlimitedSender{
		metricsService: newMetricsService(client, logger),
		logger:         logger,
	}
}

// Send отправляет метрики немедленно в новой горутине
func (us *unlimitedSender) Send(
	ctx context.Context,
	metrics model.MemoryMetrics,
	deltaCounter int64,
	logger *zap.Logger,
) error {
	us.wg.Add(1)

	go func() {
		defer us.wg.Done()

		err := us.metricsService.send(ctx, metrics, deltaCounter)
		if err != nil {
			us.logger.Error("failed to send metrics in unlimited mode", zap.Error(err))
		} else {
			us.logger.Debug("metrics sent successfully in unlimited mode")
		}
	}()

	return nil
}

// Stop ожидает завершения всех отправок
func (us *unlimitedSender) Stop() {
	us.logger.Info("waiting for all unlimited sends to complete...")
	us.wg.Wait()
	us.logger.Info("all unlimited sends completed")
}
