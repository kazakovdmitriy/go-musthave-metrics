package service

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipCompressor_CompressResponse_GzipNotAccepted(t *testing.T) {
	g := NewGzipCompressor()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	w := g.CompressResponse(recorder, req)

	_, err := w.Write([]byte("Hello, plain!"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if _, ok := w.(*compressWriter); ok {
		t.Fatal("Expected plain ResponseWriter, got compressWriter")
	}

	if recorder.Header().Get("Content-Encoding") != "" {
		t.Errorf("Expected no Content-Encoding, got %s", recorder.Header().Get("Content-Encoding"))
	}

	if recorder.Body.String() != "Hello, plain!" {
		t.Errorf("Expected 'Hello, plain!', got %q", recorder.Body.String())
	}
}

func TestGzipCompressor_CompressResponse_StatusError_NoGzipHeader(t *testing.T) {
	g := NewGzipCompressor()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := g.CompressResponse(recorder, req)
	w.WriteHeader(http.StatusNotFound)

	w.Write([]byte("Not found"))

	if cw, ok := w.(*compressWriter); ok {
		cw.Close()
	}

	if recorder.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Content-Encoding should not be set for error status")
	}
}

func TestGzipCompressor_DecompressRequest_GzipEncoded(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write([]byte("Compressed request body"))
	if err != nil {
		t.Fatalf("Failed to write to gzip: %v", err)
	}
	gw.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")

	g := NewGzipCompressor()
	err = g.DecompressRequest(req)
	if err != nil {
		t.Fatalf("DecompressRequest failed: %v", err)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("Failed to read decompressed body: %v", err)
	}
	req.Body.Close()

	if string(body) != "Compressed request body" {
		t.Errorf("Expected 'Compressed request body', got %q", string(body))
	}
}

func TestGzipCompressor_DecompressRequest_NotGzipEncoded(t *testing.T) {
	body := "Plain request body"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	g := NewGzipCompressor()
	err := g.DecompressRequest(req)
	if err != nil {
		t.Fatalf("DecompressRequest should not fail for non-gzip: %v", err)
	}

	readBody, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}
	req.Body.Close()

	if string(readBody) != body {
		t.Errorf("Expected %q, got %q", body, string(readBody))
	}
}

func TestGzipCompressor_DecompressRequest_InvalidGzip(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not gzipped"))
	req.Header.Set("Content-Encoding", "gzip")

	g := NewGzipCompressor()
	err := g.DecompressRequest(req)
	if err == nil {
		t.Fatal("Expected error for invalid gzip data")
	}
}

func TestCompressWriter_Close(t *testing.T) {
	recorder := httptest.NewRecorder()
	cw := newCompressWriter(recorder)

	_, err := cw.Write([]byte("test"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	err = cw.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	_, err = cw.Write([]byte("more"))
	if err == nil {
		t.Error("Expected error after Close")
	}
}

func TestCompressReader_Close(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write([]byte("data"))
	gw.Close()

	cr, err := newCompressReader(io.NopCloser(&buf))
	if err != nil {
		t.Fatalf("newCompressReader failed: %v", err)
	}

	_, err = io.ReadAll(cr)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	err = cr.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
