package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL           string
	httpClient        *http.Client
	headers           map[string]string
	UseGzip           bool
	CompressionLevel  int
	minSizeToCompress int
}

func NewClient(baseURL string) *Client {
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
	}
}

func (c *Client) SetHeaders(key, value string) {
	c.headers[key] = value
}

func (c *Client) SetCompression(useGzip bool, level int) {
	c.UseGzip = useGzip
	if level < gzip.DefaultCompression || level > gzip.BestCompression {
		level = gzip.DefaultCompression
	}
	c.CompressionLevel = level
}

func (c *Client) SetMinSizeToCompress(size int) {
	c.minSizeToCompress = size
}

func (c *Client) compressData(data []byte) ([]byte, error) {
	if len(data) < c.minSizeToCompress {
		return data, nil
	}

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, c.CompressionLevel)
	if err != nil {
		return nil, fmt.Errorf("creating gzip writer failed: %w", err)
	}

	if _, err := gz.Write(data); err != nil {
		return nil, fmt.Errorf("compressing data failed: %w", err)
	}

	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip writer failed: %w", err)
	}

	return buf.Bytes(), nil
}

func (c *Client) shouldCompressRequest(body []byte) bool {
	if !c.UseGzip {
		return false
	}
	if len(body) < c.minSizeToCompress {
		return false
	}
	return true
}

func (c *Client) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	var reader io.Reader
	var contentEncoding string
	var bodyData []byte

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling payload failed: %w", err)
		}
		bodyData = jsonData

		if c.shouldCompressRequest(bodyData) {
			compressedData, err := c.compressData(bodyData)
			if err != nil {
				return nil, fmt.Errorf("compressing request body failed: %w", err)
			}
			reader = bytes.NewBuffer(compressedData)
			contentEncoding = "gzip"
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

	if contentEncoding != "" {
		req.Header.Set("Content-Encoding", contentEncoding)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request failed: %w", err)
	}
	defer resp.Body.Close()

	// Читаем и при необходимости распаковываем ответ
	var respBody []byte
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("creating gzip reader for response failed: %w", err)
		}
		defer gz.Close()

		respBody, err = io.ReadAll(gz)
		if err != nil {
			return nil, fmt.Errorf("reading decompressed response failed: %w", err)
		}
	} else {
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response failed: %w", err)
		}
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *Client) Post(endpoint string, body interface{}) ([]byte, error) {
	return c.doRequest(http.MethodPost, endpoint, body)
}

func (c *Client) Get(endpoint string) ([]byte, error) {
	return c.doRequest(http.MethodGet, endpoint, nil)
}
