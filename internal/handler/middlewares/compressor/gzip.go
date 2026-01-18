package compressor

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var (
	gzipWriterPool = sync.Pool{
		New: func() interface{} {
			return gzip.NewWriter(io.Discard)
		},
	}

	gzipReaderPool = sync.Pool{
		New: func() interface{} {
			return new(gzip.Reader)
		},
	}
)

type HTTPGzipAdapter struct {
	// stream больше не нужен, если используем пулы напрямую
}

func NewHTTPGzipAdapter() *HTTPGzipAdapter {
	return &HTTPGzipAdapter{}
}

type compressWriter struct {
	w             http.ResponseWriter
	zw            *gzip.Writer
	pool          *sync.Pool
	statusCode    int
	headerWritten bool
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	// Автоматически пишем заголовок при первой записи
	if !c.headerWritten {
		c.WriteHeader(http.StatusOK)
	}
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if c.headerWritten {
		return
	}
	c.headerWritten = true
	c.statusCode = statusCode

	// Устанавливаем заголовок только для успешных ответов
	if statusCode < 300 && statusCode >= 200 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	if c.zw != nil {
		err := c.zw.Close()
		c.zw.Reset(io.Discard) // Сбрасываем перед возвратом в пул
		gzipWriterPool.Put(c.zw)
		c.zw = nil
		return err
	}
	return nil
}

type compressReader struct {
	r    io.ReadCloser
	zr   *gzip.Reader
	pool *sync.Pool
}

func (c *compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	var err1, err2 error

	if c.zr != nil {
		err1 = c.zr.Close()
		gzipReaderPool.Put(c.zr)
		c.zr = nil
	}

	if c.r != nil {
		err2 = c.r.Close()
		c.r = nil
	}

	if err1 != nil {
		return err1
	}
	return err2
}

func (g *HTTPGzipAdapter) DecompressRequest(r *http.Request) error {
	if !isGzipEncoding(r.Header.Get("Content-Encoding")) {
		return nil
	}

	// Достаем из пула или создаем новый
	zr := gzipReaderPool.Get().(*gzip.Reader)
	if err := zr.Reset(r.Body); err != nil {
		gzipReaderPool.Put(zr) // Возвращаем обратно в случае ошибки
		return err
	}

	r.Body = &compressReader{
		r:    r.Body,
		zr:   zr,
		pool: &gzipReaderPool,
	}
	return nil
}

func (g *HTTPGzipAdapter) CompressResponse(w http.ResponseWriter, r *http.Request) http.ResponseWriter {
	// Проверяем, не сжат ли уже ответ
	if w.Header().Get("Content-Encoding") != "" {
		return w
	}

	if !isGzipAccepted(r.Header.Get("Accept-Encoding")) {
		return w
	}

	// Достаем writer из пула
	zw := gzipWriterPool.Get().(*gzip.Writer)
	zw.Reset(w)

	return &compressWriter{
		w:    w,
		zw:   zw,
		pool: &gzipWriterPool,
	}
}

// Оптимизированные функции проверки заголовков
func isGzipEncoding(enc string) bool {
	// Быстрая проверка - если строка пустая или не содержит gzip
	if enc == "" || !strings.Contains(enc, "gzip") {
		return false
	}

	// Более точная проверка
	for i := 0; i < len(enc); {
		// Пропускаем пробелы и запятые
		for i < len(enc) && (enc[i] == ' ' || enc[i] == ',') {
			i++
		}

		// Проверяем "gzip"
		if i+4 <= len(enc) && strings.EqualFold(enc[i:i+4], "gzip") {
			// Проверяем, что это отдельное значение (конец строки или запятая)
			next := i + 4
			if next == len(enc) || enc[next] == ',' || enc[next] == ' ' {
				return true
			}
		}

		// Ищем следующую запятую
		for i < len(enc) && enc[i] != ',' {
			i++
		}
		i++ // Пропускаем запятую
	}
	return false
}

func isGzipAccepted(acceptEnc string) bool {
	// Можно добавить проверку качества (q=) если нужно
	return isGzipEncoding(acceptEnc)
}
