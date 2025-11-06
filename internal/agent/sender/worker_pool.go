package sender

import (
	"context"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// WorkerPool управляет пулом воркеров для отправки метрик
type WorkerPool struct {
	client         interfaces.HTTPClient
	tasks          chan SendTask
	wg             sync.WaitGroup
	logger         *zap.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	workers        int
	activeWorkers  int32
	queueSize      int
	droppedTasks   uint64
	processedTasks uint64
}

// NewWorkerPool создает новый пул воркеров
func NewWorkerPool(client interfaces.HTTPClient, workers int, queueSize int, logger *zap.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		client:    client,
		tasks:     make(chan SendTask, queueSize),
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		workers:   workers,
		queueSize: queueSize,
	}
}

// Start запускает пул воркеров
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	wp.logger.Info("worker pool started",
		zap.Int("workers", wp.workers),
		zap.Int("queue_size", wp.queueSize),
	)
}

// Stop останавливает пул воркеров
func (wp *WorkerPool) Stop() {
	wp.cancel()
	close(wp.tasks)
	wp.wg.Wait()

	wp.logger.Info("worker pool stopped",
		zap.Uint64("processed_tasks", atomic.LoadUint64(&wp.processedTasks)),
		zap.Uint64("dropped_tasks", atomic.LoadUint64(&wp.droppedTasks)),
	)
}

// GetStats возвращает статистику пула
func (wp *WorkerPool) GetStats() (activeWorkers int, queueLength int, droppedTasks uint64, processedTasks uint64) {
	return int(atomic.LoadInt32(&wp.activeWorkers)),
		len(wp.tasks),
		atomic.LoadUint64(&wp.droppedTasks),
		atomic.LoadUint64(&wp.processedTasks)
}

// Submit добавляет задачу в очередь
func (wp *WorkerPool) Submit(task SendTask) bool {
	select {
	case wp.tasks <- task:
		return true
	case <-wp.ctx.Done():
		atomic.AddUint64(&wp.droppedTasks, 1)
		return false
	default:
		atomic.AddUint64(&wp.droppedTasks, 1)

		if atomic.LoadUint64(&wp.droppedTasks)%100 == 1 {
			active, queue, dropped, processed := wp.GetStats()
			wp.logger.Warn("worker pool queue is full, dropping metrics batch",
				zap.Int("active_workers", active),
				zap.Int("queue_length", queue),
				zap.Uint64("dropped_tasks", dropped),
				zap.Uint64("processed_tasks", processed),
			)
		}
		return false
	}
}

// worker обрабатывает задачи из очереди
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	wp.logger.Debug("worker started", zap.Int("worker_id", id))

	for {
		select {
		case task, ok := <-wp.tasks:
			if !ok {
				wp.logger.Debug("worker stopping", zap.Int("worker_id", id))
				return
			}

			atomic.AddInt32(&wp.activeWorkers, 1)

			wp.logger.Debug("worker processing task",
				zap.Int("worker_id", id),
				zap.Int("metrics_count", len(task.Metrics.ToMap())),
			)

			start := time.Now()
			err := wp.sendMetrics(task.Metrics, task.Count)
			duration := time.Since(start)

			atomic.AddInt32(&wp.activeWorkers, -1)
			atomic.AddUint64(&wp.processedTasks, 1)

			if err != nil {
				wp.logger.Error("failed to send metrics",
					zap.Int("worker_id", id),
					zap.Duration("duration", duration),
					zap.Error(err),
				)
			} else {
				wp.logger.Debug("metrics sent successfully",
					zap.Int("worker_id", id),
					zap.Duration("duration", duration),
				)
			}

		case <-wp.ctx.Done():
			wp.logger.Debug("worker stopping by context", zap.Int("worker_id", id))
			return
		}
	}
}

// sendMetrics отправляет метрики на сервер
func (wp *WorkerPool) sendMetrics(metrics model.MemoryMetrics, deltaCounter int64) error {
	metricsMap := metrics.ToMap()

	if len(metricsMap) == 0 && deltaCounter == 0 {
		wp.logger.Info("no metrics to send")
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
		wp.logger.Info("no metrics to send after filtering")
		return nil
	}

	_, err := wp.client.Post(wp.ctx, "/updates/", batch)
	if err != nil {
		wp.logger.Error(
			"failed to send metrics batch",
			zap.Int("metrics_count", len(batch)),
			zap.Error(err),
		)
		return err
	}

	wp.logger.Debug(
		"successfully sent metrics batch",
		zap.Int("metrics_count", len(batch)),
	)

	return nil
}
