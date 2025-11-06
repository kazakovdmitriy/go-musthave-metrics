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
	client interfaces.HTTPClient
	logger *zap.Logger
	wg     sync.WaitGroup
}

// NewUnlimitedSender создает неограниченный отправитель
func NewUnlimitedSender(client interfaces.HTTPClient, logger *zap.Logger) interfaces.MetricsSender {
	return &unlimitedSender{
		client: client,
		logger: logger,
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

		err := us.sendMetrics(metrics, deltaCounter)
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

// sendMetrics содержит логику отправки метрик
func (us *unlimitedSender) sendMetrics(metrics model.MemoryMetrics, deltaCounter int64) error {
	metricsMap := metrics.ToMap()

	if len(metricsMap) == 0 && deltaCounter == 0 {
		us.logger.Info("no metrics to send")
		return nil
	}

	var batch []model.Metrics

	for name, value := range metricsMap {
		valueCopy := value
		batch = append(batch, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &valueCopy,
		})
	}

	if deltaCounter != 0 {
		deltaCopy := deltaCounter
		batch = append(batch, model.Metrics{
			ID:    "PollCount",
			MType: model.Counter,
			Delta: &deltaCopy,
		})
	}

	if len(batch) == 0 {
		us.logger.Info("no metrics to send after filtering")
		return nil
	}

	_, err := us.client.Post(context.Background(), "/updates/", batch)
	if err != nil {
		us.logger.Error(
			"failed to send metrics batch",
			zap.Int("metrics_count", len(batch)),
			zap.Error(err),
		)
		return err
	}

	us.logger.Debug(
		"successfully sent metrics batch",
		zap.Int("metrics_count", len(batch)),
	)

	return nil
}
