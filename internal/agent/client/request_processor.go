package client

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/middlewares/signer"
	compressorservice "github.com/kazakovdmitriy/go-musthave-metrics/internal/service/compressor_service"
)

// RequestProcessor обрабатывает запросы (подпись, сжатие)
type RequestProcessor struct {
	signer            signer.Signer
	useGzip           bool
	compressionLevel  int
	minSizeToCompress int
}

// NewRequestProcessor создает новый процессор запросов
func NewRequestProcessor(signer signer.Signer, useGzip bool, compressionLevel int) (*RequestProcessor, error) {
	if compressionLevel < gzip.DefaultCompression || compressionLevel > gzip.BestCompression {
		return nil, fmt.Errorf("compression level %d is out of valid range [%d, %d]", compressionLevel, gzip.DefaultCompression, gzip.BestCompression)
	}

	return &RequestProcessor{
		signer:            signer,
		useGzip:           useGzip,
		compressionLevel:  compressionLevel,
		minSizeToCompress: 32,
	}, nil
}

// ProcessRequest обрабатывает тело запроса
func (rp *RequestProcessor) ProcessRequest(body interface{}) (io.Reader, []byte, string, error) {
	if body == nil {
		return nil, nil, "", nil
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, nil, "", fmt.Errorf("marshaling payload failed: %w", err)
	}

	// Вычисляем подпись
	var hashValue string
	if rp.signer != nil {
		hashValue = rp.signer.Sign(jsonData)
	}

	// Сжимаем если нужно
	var reader io.Reader
	if rp.shouldCompress(jsonData) {
		compressed, err := compressorservice.Compress(jsonData, rp.compressionLevel)
		if err != nil {
			return nil, nil, "", fmt.Errorf("compressing request body failed: %w", err)
		}
		reader = bytes.NewBuffer(compressed)
	} else {
		reader = bytes.NewBuffer(jsonData)
	}

	return reader, jsonData, hashValue, nil
}

// shouldCompress проверяет нужно ли сжимать запрос
func (rp *RequestProcessor) shouldCompress(body []byte) bool {
	return rp.useGzip && len(body) >= rp.minSizeToCompress
}
