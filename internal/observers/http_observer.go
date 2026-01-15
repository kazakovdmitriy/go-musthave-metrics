package observers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/retry"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

type HTTPObserver struct {
	url      string
	log      *zap.Logger
	client   *http.Client
	retryCfg retry.RetryConfig
	mu       sync.Mutex
}

func NewHTTPObserver(url string, log *zap.Logger, cfg *config.ServerFlags) (*HTTPObserver, error) {
	retryDelays, err := cfg.GetRetryDelaysAsDuration()
	if err != nil {
		return nil, fmt.Errorf("could not parse retry delays: %w", err)
	}

	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
				DisableKeepAlives:   false,
				MaxIdleConnsPerHost: 10,
			},
		},
		retryCfg: retry.RetryConfig{
			MaxRetries: cfg.MaxRetries,
			Delays:     retryDelays,
			IsRetryableFn: func(err error) bool {
				return isNetworkError(err)
			},
		},
		log: log,
	}, nil
}

func (h *HTTPObserver) OnMetricProcessed(event model.MetricProcessedEvent) {
	go h.sendEventAsync(event)
}

func (h *HTTPObserver) sendEventAsync(event model.MetricProcessedEvent) {
	jsonData, err := json.Marshal(event)
	if err != nil {
		h.log.Error("Failed to marshal event", zap.Error(err))
		return
	}

	op := func() error {
		h.mu.Lock()
		defer h.mu.Unlock()

		req, err := http.NewRequest(http.MethodPost, h.url, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := h.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			body, _ := io.ReadAll(resp.Body)
			h.log.Debug("Server error, will retry",
				zap.Int("status", resp.StatusCode),
				zap.ByteString("body", body))
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			h.log.Warn("Client error, not retrying",
				zap.Int("status", resp.StatusCode),
				zap.ByteString("body", body))
			return nil
		}

		h.log.Debug("Successfully sent request", zap.Int("status", resp.StatusCode))
		return nil
	}

	ctx := context.Background()
	if err := retry.Do(ctx, h.retryCfg, op); err != nil {
		h.log.Error("All retries failed", zap.Error(err))
	}
}

func (h *HTTPObserver) Close() error {
	if h.client != nil {
		h.client.CloseIdleConnections()
	}
	return nil
}

// isNetworkError проверяет что ошибка сетевая
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return strings.Contains(errorStr, "connection refused") ||
		strings.Contains(errorStr, "timeout") ||
		strings.Contains(errorStr, "network")
}
