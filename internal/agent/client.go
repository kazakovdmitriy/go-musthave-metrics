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

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
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
			compressed, err := service.Compress(bodyData, c.CompressionLevel)
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
		decompressed, err := service.Decompress(rawBody)
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

func (c *Client) Post(endpoint string, body interface{}) ([]byte, error) {
	return c.doRequest(http.MethodPost, endpoint, body)
}

func (c *Client) Get(endpoint string) ([]byte, error) {
	return c.doRequest(http.MethodGet, endpoint, nil)
}

func isGzipEncoding(enc string) bool {
	for _, part := range strings.Split(enc, ",") {
		if strings.TrimSpace(strings.ToLower(part)) == "gzip" {
			return true
		}
	}
	return false
}
