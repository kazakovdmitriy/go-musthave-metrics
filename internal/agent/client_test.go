package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("should create client with http prefix when no protocol provided", func(t *testing.T) {
		client := NewClient("example.com:8080")
		expected := "http://example.com:8080"
		assert.Equal(t, expected, client.baseURL)
	})

	t.Run("should create client with localhost prefix when port only provided", func(t *testing.T) {
		client := NewClient(":8080")
		expected := "http://localhost:8080"
		assert.Equal(t, expected, client.baseURL)
	})

	t.Run("should preserve existing http protocol", func(t *testing.T) {
		client := NewClient("http://example.com:8080")
		expected := "http://example.com:8080"
		assert.Equal(t, expected, client.baseURL)
	})

	t.Run("should preserve existing https protocol", func(t *testing.T) {
		client := NewClient("https://example.com:8080")
		expected := "https://example.com:8080"
		assert.Equal(t, expected, client.baseURL)
	})

	t.Run("should initialize default values", func(t *testing.T) {
		client := NewClient("http://example.com")
		assert.True(t, client.UseGzip)
		assert.Equal(t, 32, client.minSizeToCompress)
		assert.Equal(t, gzip.DefaultCompression, client.CompressionLevel)
	})
}

func TestClient_SetHeaders(t *testing.T) {
	client := NewClient("http://example.com")
	client.SetHeaders("Authorization", "Bearer token123")
	client.SetHeaders("X-Custom", "value")

	assert.Equal(t, "Bearer token123", client.headers["Authorization"])
	assert.Equal(t, "value", client.headers["X-Custom"])
}

func TestClient_SetCompression(t *testing.T) {
	client := NewClient("http://example.com")

	t.Run("should disable compression when useGzip is false", func(t *testing.T) {
		client.SetCompression(false, 6)
		assert.False(t, client.UseGzip)
	})

	t.Run("should set valid compression level", func(t *testing.T) {
		client.SetCompression(true, gzip.BestCompression)
		assert.Equal(t, gzip.BestCompression, client.CompressionLevel)
	})

	t.Run("should reset to default when invalid level provided", func(t *testing.T) {
		client.SetCompression(true, -100)
		assert.Equal(t, gzip.DefaultCompression, client.CompressionLevel)

		client.SetCompression(true, 100)
		assert.Equal(t, gzip.DefaultCompression, client.CompressionLevel)
	})
}

func TestClient_SetMinSizeToCompress(t *testing.T) {
	client := NewClient("http://example.com")
	client.SetMinSizeToCompress(100)
	assert.Equal(t, 100, client.minSizeToCompress)
}

func TestClient_compressData(t *testing.T) {
	client := NewClient("http://example.com")

	t.Run("should compress data larger than minSizeToCompress", func(t *testing.T) {
		data := []byte(strings.Repeat("a", 100))
		compressed, err := client.compressData(data)
		require.NoError(t, err)
		assert.Less(t, len(compressed), len(data))

		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		require.NoError(t, err)
		decompressed, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, data, decompressed)
	})

	t.Run("should not compress data smaller than minSizeToCompress", func(t *testing.T) {
		data := []byte("small")
		compressed, err := client.compressData(data)
		require.NoError(t, err)
		assert.Equal(t, data, compressed)
	})

	t.Run("should handle compression errors", func(t *testing.T) {
		client.CompressionLevel = -100 // invalid
		data := []byte(strings.Repeat("a", 100))
		_, err := client.compressData(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "creating gzip writer failed")
	})
}

func TestClient_shouldCompressRequest(t *testing.T) {
	client := NewClient("http://example.com")

	t.Run("should return false when compression is disabled", func(t *testing.T) {
		client.UseGzip = false
		result := client.shouldCompressRequest([]byte(strings.Repeat("a", 100)))
		assert.False(t, result)
	})

	t.Run("should return false when data is smaller than minSizeToCompress", func(t *testing.T) {
		client.UseGzip = true
		result := client.shouldCompressRequest([]byte("small"))
		assert.False(t, result)
	})

	t.Run("should return true when compression is enabled and data is large enough", func(t *testing.T) {
		client.UseGzip = true
		result := client.shouldCompressRequest([]byte(strings.Repeat("a", 100)))
		assert.True(t, result)
	})
}

func TestClient_Post(t *testing.T) {
	t.Run("should send POST request with JSON body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "gzip", r.Header.Get("Accept-Encoding"))

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var data map[string]interface{}
			err = json.Unmarshal(body, &data)
			require.NoError(t, err)
			assert.Equal(t, "test", data["name"])

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		response, err := client.Post("/test", map[string]string{"name": "test"})
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(response, &result)
		require.NoError(t, err)
		assert.Equal(t, "ok", result["status"])
	})

	t.Run("should handle compressed request body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))

			gz, err := gzip.NewReader(r.Body)
			require.NoError(t, err)
			defer gz.Close()

			body, err := io.ReadAll(gz)
			require.NoError(t, err)

			var data map[string]interface{}
			err = json.Unmarshal(body, &data)
			require.NoError(t, err)
			assert.Equal(t, "large_data", data["name"])

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "compressed_ok"}`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		largeData := map[string]string{
			"name": "large_data",
			"data": strings.Repeat("x", 100),
		}
		response, err := client.Post("/test", largeData)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(response, &result)
		require.NoError(t, err)
		assert.Equal(t, "compressed_ok", result["status"])
	})

	t.Run("should handle compressed response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			gz.Write([]byte(`{"compressed": true, "data": "response_data"}`))
			gz.Close()

			w.Write(buf.Bytes())
		}))
		defer server.Close()

		client := NewClient(server.URL)
		response, err := client.Post("/test", map[string]string{"test": "data"})
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(response, &result)
		require.NoError(t, err)
		assert.Equal(t, true, result["compressed"])
		assert.Equal(t, "response_data", result["data"])
	})

	t.Run("should handle request errors", func(t *testing.T) {
		client := NewClient("http://invalid-url-that-does-not-exist:9999")
		_, err := client.Post("/test", map[string]string{"test": "data"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "executing request failed")
	})

	t.Run("should handle HTTP error status codes", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		_, err := client.Post("/nonexistent", map[string]string{"test": "data"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request failed with status 404")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("should send custom headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
			assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "headers_received"}`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHeaders("Authorization", "Bearer token123")
		client.SetHeaders("X-Custom-Header", "custom-value")

		_, err := client.Post("/test", map[string]string{"test": "data"})
		require.NoError(t, err)
	})
}

func TestClient_Get(t *testing.T) {
	t.Run("should send GET request without body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "", r.Header.Get("Content-Encoding"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"method": "GET", "status": "ok"}`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		response, err := client.Get("/test")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(response, &result)
		require.NoError(t, err)
		assert.Equal(t, "GET", result["method"])
		assert.Equal(t, "ok", result["status"])
	})

	t.Run("should handle compressed GET response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			gz.Write([]byte(`{"method": "GET", "compressed": true}`))
			gz.Close()

			w.Write(buf.Bytes())
		}))
		defer server.Close()

		client := NewClient(server.URL)
		response, err := client.Get("/test")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(response, &result)
		require.NoError(t, err)
		assert.Equal(t, "GET", result["method"])
		assert.Equal(t, true, result["compressed"])
	})

	t.Run("should handle GET request errors", func(t *testing.T) {
		client := NewClient("http://invalid-url-get:9999")
		_, err := client.Get("/test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "executing request failed")
	})
}

func TestClient_MarshalingErrors(t *testing.T) {
	t.Run("should handle JSON marshaling errors", func(t *testing.T) {
		client := NewClient("http://example.com")

		ch := make(chan int)
		_, err := client.Post("/test", ch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "marshaling payload failed")
	})
}

func TestClient_Timeout(t *testing.T) {
	t.Run("should handle timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.httpClient.Timeout = 50 * time.Millisecond

		_, err := client.Post("/test", map[string]string{"test": "data"})
		assert.Error(t, err)
	})
}

func TestClient_WithDisabledCompression(t *testing.T) {
	t.Run("should not compress when disabled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "", r.Header.Get("Content-Encoding"))

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var data map[string]interface{}
			err = json.Unmarshal(body, &data)
			require.NoError(t, err)
			assert.Equal(t, "test", data["name"])

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "no_compression"}`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetCompression(false, 0)

		response, err := client.Post("/test", map[string]string{"name": "test"})
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(response, &result)
		require.NoError(t, err)
		assert.Equal(t, "no_compression", result["status"])
	})
}

func TestClient_WithCustomMinSize(t *testing.T) {
	t.Run("should respect custom min size to compress", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hasCompression := r.Header.Get("Content-Encoding") == "gzip"

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			if hasCompression {
				gz, err := gzip.NewReader(bytes.NewReader(body))
				require.NoError(t, err)
				defer gz.Close()
				body, err = io.ReadAll(gz)
				require.NoError(t, err)
			}

			var data map[string]interface{}
			err = json.Unmarshal(body, &data)
			require.NoError(t, err)
			assert.Equal(t, "test", data["name"])

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"compressed": ` + fmt.Sprintf("%t", hasCompression) + `}`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetMinSizeToCompress(1000)

		response, err := client.Post("/test", map[string]string{"name": "test"})
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(response, &result)
		require.NoError(t, err)
	})
}
