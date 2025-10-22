package compressor

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	compressorservice "github.com/kazakovdmitriy/go-musthave-metrics/internal/service/compressor_service"
)

// HTTPGzipAdapter — адаптер для HTTP
type HTTPGzipAdapter struct {
	stream *compressorservice.StreamCompressor
}

func NewHTTPGzipAdapter() *HTTPGzipAdapter {
	return &HTTPGzipAdapter{
		stream: compressorservice.NewStreamCompressor(),
	}
}

// compressWriter — обёртка для сжатия ответа
type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	return c.zw.Close()
}

// compressReader — обёртка для распаковки запроса
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func (c *compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	err1 := c.r.Close()
	err2 := c.zr.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

// DecompressRequest распаковывает тело запроса, если оно сжато
func (g *HTTPGzipAdapter) DecompressRequest(r *http.Request) error {
	if !isGzipEncoding(r.Header.Get("Content-Encoding")) {
		return nil
	}
	zr, err := g.stream.NewReader(r.Body)
	if err != nil {
		return err
	}
	r.Body = &compressReader{r: r.Body, zr: zr}
	return nil
}

// CompressResponse оборачивает ResponseWriter для сжатия
func (g *HTTPGzipAdapter) CompressResponse(w http.ResponseWriter, r *http.Request) http.ResponseWriter {
	if isGzipAccepted(r.Header.Get("Accept-Encoding")) {
		zw := g.stream.NewWriter(w)
		return &compressWriter{w: w, zw: zw}
	}
	return w
}

// Вспомогательные функции парсинга заголовков
func isGzipEncoding(enc string) bool {
	for _, e := range strings.Split(enc, ",") {
		if strings.TrimSpace(strings.ToLower(e)) == "gzip" {
			return true
		}
	}
	return false
}

func isGzipAccepted(acceptEnc string) bool {
	return isGzipEncoding(acceptEnc)
}
