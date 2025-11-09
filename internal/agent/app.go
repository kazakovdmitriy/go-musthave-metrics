package agent

import (
	"context"
	"sync"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/client"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/collector"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/provider"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/reporter"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/sender"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares/signer"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service/signerservice"
	"go.uber.org/zap"
)

// App представляет основное приложение агента
type App struct {
	config    *config.AgentFlags
	logger    *zap.Logger
	collector interfaces.MetricsCollector
	reporter  interfaces.MetricsReporter
	wg        sync.WaitGroup
}

// Initialize создает и настраивает все компоненты приложения
func (a *App) Initialize(ctx context.Context) error {
	a.logger.Info("initializing agent",
		zap.Int("rate_limit", a.config.RateLimit),
		zap.Bool("rate_limiting_enabled", a.config.RateLimit > 0),
	)

	var signerService signer.Signer
	if a.config.SecretKey != "" {
		signerService = signerservice.NewSHA256Signer(a.config.SecretKey)
	}

	httpClient := client.NewClient(
		a.config.ServerAddr,
		signerService,
		a.logger,
		a.config,
	)

	providers := []interfaces.MetricsProvider{
		provider.NewRuntimeMetricsProvider(),
		provider.NewGopsutilProvider(),
	}

	pollingInterval := time.Duration(a.config.PollingInterval) * time.Second
	reportInterval := time.Duration(a.config.ReportInterval) * time.Second

	a.collector = collector.NewMetricsCollector(ctx, pollingInterval, a.logger, providers)

	var metricsSender interfaces.MetricsSender

	if a.config.RateLimit > 0 {
		// Ограниченный режим с worker pool
		metricsSender = sender.NewMetricsSender(
			httpClient,
			a.config.RateLimit,
			a.config.RateLimit*2,
			a.logger,
		)
		a.logger.Info("using limited sender with worker pool",
			zap.Int("workers", a.config.RateLimit),
			zap.Int("queue_size", a.config.RateLimit*2),
		)
	} else {
		// Неограниченный режим
		metricsSender = sender.NewMetricsSender(
			httpClient,
			0,
			0,
			a.logger,
		)
		a.logger.Info("using unlimited sender")
	}

	// Создаем репортер
	a.reporter = reporter.NewMetricsReporter(ctx, a.collector, metricsSender, reportInterval, a.logger)

	return nil
}

// NewAppWithConfig создает новое приложение и инициализирует его
func NewAppWithConfig(ctx context.Context, cfg *config.AgentFlags, logger *zap.Logger) *App {
	app := NewApp(cfg, nil, nil, logger)
	if err := app.Initialize(ctx); err != nil {

		logger.Fatal("failed to initialize app", zap.Error(err))
	}
	return app
}

// NewApp создает новое приложение агента
func NewApp(
	cfg *config.AgentFlags,
	collector interfaces.MetricsCollector,
	reporter interfaces.MetricsReporter,
	logger *zap.Logger,
) *App {
	return &App{
		config:    cfg,
		logger:    logger,
		collector: collector,
		reporter:  reporter,
	}
}

// Run запускает приложение
func (a *App) Run(ctx context.Context) {
	a.logger.Info("starting agent components...")

	a.collector.Start()
	a.reporter.Start()

	a.logger.Info("agent started successfully",
		zap.String("server_addr", a.config.ServerAddr),
		zap.Int("poll_interval", a.config.PollingInterval),
		zap.Int("report_interval", a.config.ReportInterval),
		zap.Int("rate_limit", a.config.RateLimit),
	)

	<-ctx.Done()
	a.logger.Info("received shutdown signal, stopping agent...")
}

// Wait останавливает приложение
func (a *App) Wait() {
	a.logger.Info("waiting for agent components to stop...")

	if a.collector != nil {
		a.collector.Stop()
	}

	if a.reporter != nil {
		a.reporter.Stop()
	}

	a.logger.Info("agent shutdown completed")
}
