package compressor_service

import (
	"bytes"
	"compress/gzip"
	"io"
)

func Compress(data []byte, level int) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Decompress(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

type StreamCompressor struct{}

func NewStreamCompressor() *StreamCompressor {
	return &StreamCompressor{}
}

func (s *StreamCompressor) NewWriter(w io.Writer) *gzip.Writer {
	return gzip.NewWriter(w)
}

func (s *StreamCompressor) NewReader(r io.Reader) (*gzip.Reader, error) {
	return gzip.NewReader(r)
}
