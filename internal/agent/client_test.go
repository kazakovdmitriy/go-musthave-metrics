package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewClient(t *testing.T) {
	logger := zaptest.NewLogger(t)
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "port only",
			baseURL:  ":8080",
			expected: "http://localhost:8080",
		},
		{
			name:     "host only",
			baseURL:  "example.com",
			expected: "http://example.com",
		},
		{
			name:     "http url",
			baseURL:  "http://example.com",
			expected: "http://example.com",
		},
		{
			name:     "https url",
			baseURL:  "https://example.com    ",
			expected: "https://example.com    ",
		},
		{
			name:     "with path",
			baseURL:  "http://example.com/api",
			expected: "http://example.com/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL, logger)
			assert.Equal(t, tt.expected, client.baseURL)
			assert.NotNil(t, client.httpClient)
			assert.Equal(t, time.Second*20, client.httpClient.Timeout)
			assert.True(t, client.UseGzip)
			assert.Equal(t, gzip.DefaultCompression, client.CompressionLevel)
			assert.Equal(t, 32, client.minSizeToCompress)
		})
	}
}

func TestClient_SetHeaders(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := NewClient("http://example.com", logger)
	client.SetHeaders("Authorization", "Bearer token")
	client.SetHeaders("X-Custom", "value")

	assert.Equal(t, "Bearer token", client.headers["Authorization"])
	assert.Equal(t, "value", client.headers["X-Custom"])
}

func TestClient_SetCompression(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := NewClient("http://example.com", logger)

	// Test valid compression level
	client.SetCompression(true, gzip.BestCompression)
	assert.True(t, client.UseGzip)
	assert.Equal(t, gzip.BestCompression, client.CompressionLevel)

	// Test invalid compression level (should default to DefaultCompression)
	client.SetCompression(true, 100) // invalid level
	assert.True(t, client.UseGzip)
	assert.Equal(t, gzip.DefaultCompression, client.CompressionLevel)

	// Test disabling compression
	client.SetCompression(false, gzip.DefaultCompression)
	assert.False(t, client.UseGzip)
}

func TestClient_SetMinSizeToCompress(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := NewClient("http://example.com", logger)
	client.SetMinSizeToCompress(100)

	assert.Equal(t, 100, client.minSizeToCompress)
}

func TestClient_shouldCompressRequest(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := NewClient("http://example.com", logger)

	// Test with compression disabled
	client.UseGzip = false
	assert.False(t, client.shouldCompressRequest([]byte("small data")))
	assert.False(t, client.shouldCompressRequest(make([]byte, 100)))

	// Test with compression enabled but small data
	client.UseGzip = true
	client.minSizeToCompress = 32
	assert.False(t, client.shouldCompressRequest([]byte("small")))
	assert.True(t, client.shouldCompressRequest(make([]byte, 32)))
	assert.True(t, client.shouldCompressRequest(make([]byte, 100)))
}

func TestClient_Post(t *testing.T) {
	logger := zaptest.NewLogger(t)
	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	expectedData := testData{Name: "test", Value: 42}

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var receivedData testData
		err = json.Unmarshal(body, &receivedData)
		require.NoError(t, err)
		assert.Equal(t, expectedData, receivedData)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "created"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, logger)

	resp, err := client.Post(context.Background(), "/test", expectedData)
	require.NoError(t, err)
	assert.Equal(t, `{"status": "created"}`, string(resp))
}

func TestClient_PostWithCompression(t *testing.T) {
	logger := zaptest.NewLogger(t)
	type testData struct {
		Data string `json:"data"`
	}

	largeData := testData{Data: strings.Repeat("x", 100)} // Larger than minSizeToCompress

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))

		// Check if body is compressed
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		// Decompress the body to verify content
		gz, err := gzip.NewReader(bytes.NewReader(body))
		require.NoError(t, err)
		defer gz.Close()

		decompressed, err := io.ReadAll(gz)
		require.NoError(t, err)

		var receivedData testData
		err = json.Unmarshal(decompressed, &receivedData)
		require.NoError(t, err)
		assert.Equal(t, largeData, receivedData)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "compressed_ok"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, logger)
	client.SetMinSizeToCompress(50) // Make sure our data will be compressed

	resp, err := client.Post(context.Background(), "/test", largeData)
	require.NoError(t, err)
	assert.Equal(t, `{"status": "compressed_ok"}`, string(resp))
}

func TestClient_PostWithCustomHeaders(t *testing.T) {
	logger := zaptest.NewLogger(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "headers_ok"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, logger)
	client.SetHeaders("Authorization", "Bearer test-token")
	client.SetHeaders("X-Custom-Header", "custom-value")

	// Исправленный вызов: добавлен context
	resp, err := client.Post(context.Background(), "/test", map[string]string{"key": "value"})
	require.NoError(t, err)
	assert.Equal(t, `{"status": "headers_ok"}`, string(resp))
}

func TestClient_ErrorCases(t *testing.T) {
	logger := zaptest.NewLogger(t)
	t.Run("server returns error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "internal server error"}`))
		}))
		defer server.Close()

		client := NewClient(server.URL, logger)

		_, err := client.Get(context.Background(), "/error")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request failed with status 500")
		assert.Contains(t, err.Error(), "internal server error")
	})

	t.Run("invalid json in request", func(t *testing.T) {
		client := NewClient("http://localhost:9999", logger) // Non-existent server

		// Create a struct with unmarshalable data
		// Note: This will cause json.Marshal to panic, not return an error.
		// To test marshaling errors, avoid unmarshalable types like chan.
		// Let's use a type that *can* be marshaled but causes an error later if needed.
		// Actually, `chan int` inside a struct field will cause json.Marshal to return an error.
		// The original test was correct in intent, but Go's json.Marshal panics on chan.
		// To simulate a marshaling error, we can pass a nil writer or use reflect, but simplest is to avoid chan.
		// Let's just test the marshaling error scenario by passing a function, which json.Marshal also cannot handle.
		// Actually, json.Marshal does NOT panic on `chan int` directly if it's a top-level value passed to Marshal,
		// but it panics when it encounters unserializable types during reflection.
		// The safest way to test this is to avoid the problematic type or mock the Marshal function.
		// For simplicity here, we'll assume the original intent was to pass something json.Marshal can't handle.
		// Using `func() {}` will cause json.Marshal to return an error.
		invalidData := struct {
			Fn func() `json:"fn"` // Functions cannot be marshaled
		}{Fn: func() {}}

		_, err := client.Post(context.Background(), "/test", invalidData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "marshaling payload failed")
	})

	t.Run("network error", func(t *testing.T) {
		client := NewClient("http://localhost:9999", logger) // Non-existent server

		// Исправленный вызов: добавлен context
		_, err := client.Get(context.Background(), "/test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "executing request failed")
	})
}

func TestIsGzipEncoding(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		expected bool
	}{
		{
			name:     "single gzip",
			encoding: "gzip",
			expected: true,
		},
		{
			name:     "gzip with other encodings",
			encoding: "deflate, gzip, br",
			expected: true,
		},
		{
			name:     "case insensitive",
			encoding: "GZIP",
			expected: true,
		},
		{
			name:     "not gzip",
			encoding: "deflate",
			expected: false,
		},
		{
			name:     "empty string",
			encoding: "",
			expected: false,
		},
		{
			name:     "whitespace",
			encoding: "  gzip  ",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGzipEncoding(tt.encoding)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_PostWithEmptyBody(t *testing.T) {
	logger := zaptest.NewLogger(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "", r.Header.Get("Content-Encoding")) // Should not have compression header

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Empty(t, body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "empty_ok"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, logger)

	resp, err := client.Post(context.Background(), "/test", nil)
	require.NoError(t, err)
	assert.Equal(t, `{"status": "empty_ok"}`, string(resp))
}

func TestClient_CompressionDisabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	data := map[string]interface{}{"large": strings.Repeat("x", 100)}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "", r.Header.Get("Content-Encoding")) // Should not have compression header

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var receivedData map[string]interface{}
		err = json.Unmarshal(body, &receivedData)
		require.NoError(t, err)
		assert.Equal(t, data, receivedData)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "no_compression"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, logger)
	client.SetCompression(false, gzip.DefaultCompression)

	resp, err := client.Post(context.Background(), "/test", data)
	require.NoError(t, err)
	assert.Equal(t, `{"status": "no_compression"}`, string(resp))
}
