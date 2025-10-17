package service

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type GzipCompressor struct{}

func NewGzipCompressor() *GzipCompressor {
	return &GzipCompressor{}
}

// compressWriter обертка для сжатия ответа
type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
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

// compressReader обертка для распаковки запроса
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func (g *GzipCompressor) CompressResponse(w http.ResponseWriter, r *http.Request) http.ResponseWriter {
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		return newCompressWriter(w)
	}
	return w
}

func (g *GzipCompressor) DecompressRequest(r *http.Request) error {
	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		cr, err := newCompressReader(r.Body)
		if err != nil {
			return err
		}
		r.Body = cr
	}
	return nil
}
