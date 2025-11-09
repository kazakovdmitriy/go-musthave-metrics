package sender

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Task - функция для выполнения воркером
type Task func() error

// WorkerPool управляет пулом воркеров для ЛЮБЫХ задач
type WorkerPool struct {
	tasks          chan Task
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
func NewWorkerPool(workers int, queueSize int, logger *zap.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		tasks:     make(chan Task, queueSize),
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

// Submit добавляет задачу в очередь
func (wp *WorkerPool) Submit(task Task) bool {
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
			wp.logger.Warn("worker pool queue is full, dropping task",
				zap.Int("active_workers", active),
				zap.Int("queue_length", queue),
				zap.Uint64("dropped_tasks", dropped),
				zap.Uint64("processed_tasks", processed),
			)
		}
		return false
	}
}

// GetStats возвращает статистику пула
func (wp *WorkerPool) GetStats() (activeWorkers int, queueLength int, droppedTasks uint64, processedTasks uint64) {
	return int(atomic.LoadInt32(&wp.activeWorkers)),
		len(wp.tasks),
		atomic.LoadUint64(&wp.droppedTasks),
		atomic.LoadUint64(&wp.processedTasks)
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

			start := time.Now()
			err := task()
			duration := time.Since(start)

			atomic.AddInt32(&wp.activeWorkers, -1)
			atomic.AddUint64(&wp.processedTasks, 1)

			if err != nil {
				wp.logger.Error("task execution failed",
					zap.Int("worker_id", id),
					zap.Duration("duration", duration),
					zap.Error(err),
				)
			} else {
				wp.logger.Debug("task completed successfully",
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
