package client

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"golang.org/x/sync/semaphore"
	"net/http"
	"strings"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares/signer"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/retry"
	"go.uber.org/zap"
)

// Client реализация HTTP клиента с ограничением одновременных запросов
type Client struct {
	baseURL           string
	httpClient        *http.Client
	headers           map[string]string
	requestProcessor  *RequestProcessor
	responseProcessor *ResponseProcessor
	logger            *zap.Logger
	semaphore         *semaphore.Weighted
	cfg               *config.AgentFlags
}

// NewClient создает новый клиент с ограничением или без
func NewClient(
	baseURL string,
	signer signer.Signer,
	logger *zap.Logger,
	cfg *config.AgentFlags,
) interfaces.HTTPClient {
	baseURL = normalizeURL(baseURL)

	var sem *semaphore.Weighted
	if cfg.RateLimit > 0 {
		sem = semaphore.NewWeighted(int64(cfg.RateLimit))
		logger.Info("client semaphore initialized",
			zap.Int("rate_limit", cfg.RateLimit),
		)
	} else if cfg.RateLimit == 0 {
		logger.Info("rate limiting disabled")
	} else {
		logger.Warn("invalid rate limit, using unlimited mode", zap.Int("rate_limit", cfg.RateLimit))
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Second * 20,
		},
		headers:           make(map[string]string),
		requestProcessor:  NewRequestProcessor(signer, true, gzip.DefaultCompression),
		responseProcessor: &ResponseProcessor{},
		logger:            logger,
		semaphore:         sem,
		cfg:               cfg,
	}
}

// normalizeURL нормализует URL
func normalizeURL(url string) string {
	if strings.HasPrefix(url, ":") {
		url = "localhost" + url
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}
	return url
}

// SetHeader устанавливает заголовок
func (c *Client) SetHeader(key, value string) {
	c.headers[key] = value
}

// SetCompression настраивает сжатие
func (c *Client) SetCompression(useGzip bool, level int) {
	c.requestProcessor = NewRequestProcessor(c.requestProcessor.signer, useGzip, level)
}

// doRequest выполняет HTTP запрос
func (c *Client) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	reader, bodyData, hashValue, err := c.requestProcessor.ProcessRequest(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("creating request failed: %w", err)
	}

	c.setRequestHeaders(req, bodyData, hashValue)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := c.responseProcessor.ProcessResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// setRequestHeaders устанавливает заголовки запроса
func (c *Client) setRequestHeaders(req *http.Request, bodyData []byte, hashValue string) {
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")

	if bodyData != nil && c.requestProcessor.shouldCompress(bodyData) {
		req.Header.Set("Content-Encoding", "gzip")
	}

	if bodyData != nil && c.requestProcessor.signer != nil {
		req.Header.Set("HashSHA256", hashValue)
	}
}

// doRequestWithRetry выполняет запрос с ограничением или без
func (c *Client) doRequestWithRetry(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	// Ограничиваем количество одновременных запросов только если семафор включен
	if c.semaphore != nil {
		if err := c.semaphore.Acquire(ctx, 1); err != nil {
			return nil, fmt.Errorf("failed to acquire semaphore: %w", err)
		}
		defer c.semaphore.Release(1)
	}

	var response []byte

	retryDelays, err := c.cfg.GetRetryDelaysAsDuration()
	if err != nil {
		return nil, err
	}

	cfg := retry.RetryConfig{
		MaxRetries: c.cfg.MaxRetries,
		Delays:     retryDelays,
		IsRetryableFn: func(err error) bool {
			return isNetworkError(err)
		},
	}

	err = retry.Do(ctx, cfg, func() error {
		resp, err := c.doRequest(method, endpoint, body)
		if err != nil {
			return err
		}
		response = resp
		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

// Post выполняет POST запрос
func (c *Client) Post(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
	return c.doRequestWithRetry(ctx, http.MethodPost, endpoint, body)
}

// Get выполняет GET запрос
func (c *Client) Get(ctx context.Context, endpoint string) ([]byte, error) {
	return c.doRequestWithRetry(ctx, http.MethodGet, endpoint, nil)
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
