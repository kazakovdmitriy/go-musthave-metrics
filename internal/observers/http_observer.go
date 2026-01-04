package observers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

type HTTPObserver struct {
	url    string
	log    *zap.Logger
	client *http.Client
	mu     sync.Mutex
}

func NewHTTPObserver(url string, log *zap.Logger) *HTTPObserver {
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
		log: log,
	}
}

func (h *HTTPObserver) OnMetricProcessed(event model.MetricProcessedEvent) {
	go h.sendEventAsync(event)
}

func (h *HTTPObserver) sendEventAsync(event model.MetricProcessedEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	jsonData, err := json.Marshal(event)
	if err != nil {
		h.log.Error("Failed to marshal event", zap.Error(err))
		return
	}

	req, err := http.NewRequest(http.MethodPost, h.url, bytes.NewBuffer(jsonData))
	if err != nil {
		h.log.Error("Failed to create request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		h.log.Warn("Failed to send request", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		h.log.Warn("Failed to send request", zap.Int("status", resp.StatusCode), zap.ByteString("body", body))
	} else {
		h.log.Debug("Successfully sent request", zap.Int("status", resp.StatusCode))
	}
}
