package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	compressorservice "github.com/kazakovdmitriy/go-musthave-metrics/internal/service/compressor_service"
	"go.uber.org/zap"
)

type Client struct {
	baseURL           string
	httpClient        *http.Client
	headers           map[string]string
	UseGzip           bool
	CompressionLevel  int
	minSizeToCompress int
	log               *zap.Logger
}

func NewClient(baseURL string, log *zap.Logger) *Client {
	if strings.HasPrefix(baseURL, ":") {
		baseURL = "localhost" + baseURL
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Second * 20,
		},
		headers:           make(map[string]string),
		UseGzip:           true,
		CompressionLevel:  gzip.DefaultCompression,
		minSizeToCompress: 32,
		log:               log,
	}
}

func (c *Client) SetHeaders(key, value string) {
	c.headers[key] = value
}

func (c *Client) SetCompression(useGzip bool, level int) {
	c.UseGzip = useGzip
	if level < gzip.DefaultCompression || level > gzip.BestCompression {
		level = gzip.DefaultCompression
		log.Printf("Compression level %d is out of valid range [%d, %d], using default level %d",
			level, gzip.DefaultCompression, gzip.BestCompression, gzip.DefaultCompression)
	}
	c.CompressionLevel = level
	log.Printf("Compression settings updated: UseGzip=%v, Level=%d", useGzip, level)
}

func (c *Client) SetMinSizeToCompress(size int) {
	c.minSizeToCompress = size
}

func (c *Client) shouldCompressRequest(body []byte) bool {
	return c.UseGzip && len(body) >= c.minSizeToCompress
}

func (c *Client) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	var reader io.Reader
	var bodyData []byte

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling payload failed: %w", err)
		}
		bodyData = jsonData

		if c.shouldCompressRequest(bodyData) {
			compressed, err := compressorservice.Compress(bodyData, c.CompressionLevel)
			if err != nil {
				return nil, fmt.Errorf("compressing request body failed: %w", err)
			}
			reader = bytes.NewBuffer(compressed)
		} else {
			reader = bytes.NewBuffer(bodyData)
		}
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("creating request failed: %w", err)
	}

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")

	if body != nil && c.shouldCompressRequest(bodyData) {
		req.Header.Set("Content-Encoding", "gzip")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body failed: %w", err)
	}

	var finalBody []byte
	if isGzipEncoding(resp.Header.Get("Content-Encoding")) {
		decompressed, err := compressorservice.Decompress(rawBody)
		if err != nil {
			return nil, fmt.Errorf("decompressing response body failed: %w", err)
		}
		finalBody = decompressed
	} else {
		finalBody = rawBody
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(finalBody))
	}

	return finalBody, nil
}

func (c *Client) doRequestWithRetry(method, endpoint string, body interface{}) ([]byte, error) {
	var lastErr error
	intervals := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	attempts := len(intervals) + 1

	for i := 0; i < attempts; i++ {
		resp, err := c.doRequest(method, endpoint, body)
		if err == nil {
			return resp, err
		}

		if !isNetworkError(err) {
			return nil, err
		}

		lastErr = err

		if i < len(intervals) {
			c.log.Info(
				"Retrying request due to network error",
				zap.String("method", method),
				zap.String("endpoint", endpoint),
				zap.Int("attempt", i+1),
				zap.Duration("sleep", intervals[i]),
				zap.Error(err),
			)
			time.Sleep(intervals[i])
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", attempts, lastErr)
}

func isNetworkError(err error) bool {
	return strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "network")
}

func (c *Client) Post(endpoint string, body interface{}) ([]byte, error) {
	return c.doRequestWithRetry(http.MethodPost, endpoint, body)
}

func (c *Client) Get(endpoint string) ([]byte, error) {
	return c.doRequestWithRetry(http.MethodGet, endpoint, nil)
}

func isGzipEncoding(enc string) bool {
	for _, part := range strings.Split(enc, ",") {
		if strings.TrimSpace(strings.ToLower(part)) == "gzip" {
			return true
		}
	}
	return false
}
