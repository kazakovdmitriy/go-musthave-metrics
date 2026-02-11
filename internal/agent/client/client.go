package client

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/utils/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/utils/crypto"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/utils/retry"
	"net/http"
	"strings"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent/interfaces"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares/signer"
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
	cfg               *config.AgentFlags
	cryptoService     *crypto.CryptoService
}

// NewClient создает новый клиент с ограничением или без
func NewClient(
	baseURL string,
	signer signer.Signer,
	logger *zap.Logger,
	cfg *config.AgentFlags,
) (interfaces.HTTPClient, error) {
	baseURL = normalizeURL(baseURL)

	var cryptoService *crypto.CryptoService
	if cfg.CryptoKeyPath != "" {
		var err error
		cryptoService, err = crypto.NewCryptoService(cfg.CryptoKeyPath, true) // true = агент
		if err != nil {
			logger.Warn("crypto: failed to initialize", zap.Error(err), zap.String("path", cfg.CryptoKeyPath))
		} else if cryptoService.IsEnabled() {
			logger.Info("crypto: encryption enabled", zap.String("path", cfg.CryptoKeyPath))
		}
	}

	requestProcessor, err := NewRequestProcessor(
		signer,
		true,
		gzip.DefaultCompression,
		cryptoService,
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Second * 20,
		},
		headers:           make(map[string]string),
		requestProcessor:  requestProcessor,
		responseProcessor: &ResponseProcessor{},
		logger:            logger,
		cfg:               cfg,
		cryptoService:     cryptoService,
	}, nil
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
