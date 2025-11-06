package client

import (
	"fmt"
	compressorservice "github.com/kazakovdmitriy/go-musthave-metrics/internal/service/compressor_service"
	"io"
	"net/http"
	"strings"
)

// ResponseProcessor обрабатывает ответы
type ResponseProcessor struct{}

// ProcessResponse обрабатывает тело ответа
func (rp *ResponseProcessor) ProcessResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body failed: %w", err)
	}

	if isGzipEncoding(resp.Header.Get("Content-Encoding")) {
		decompressed, err := compressorservice.Decompress(rawBody)
		if err != nil {
			return nil, fmt.Errorf("decompressing response body failed: %w", err)
		}
		return decompressed, nil
	}

	return rawBody, nil
}

// isGzipEncoding проверяет gzip кодировку
func isGzipEncoding(enc string) bool {
	for _, part := range strings.Split(enc, ",") {
		if strings.TrimSpace(strings.ToLower(part)) == "gzip" {
			return true
		}
	}
	return false
}
