package service

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"testing"
)

func TestCompress(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		level   int
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    []byte{},
			level:   gzip.DefaultCompression,
			wantErr: false,
		},
		{
			name:    "normal data",
			data:    []byte("hello world"),
			level:   gzip.DefaultCompression,
			wantErr: false,
		},
		{
			name:    "best speed",
			data:    []byte("test data for compression"),
			level:   gzip.BestSpeed,
			wantErr: false,
		},
		{
			name:    "best compression",
			data:    []byte("test data for compression"),
			level:   gzip.BestCompression,
			wantErr: false,
		},
		{
			name:    "invalid compression level",
			data:    []byte("test"),
			level:   10, // invalid level
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Compress(tt.data, tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Verify that decompression works correctly
			decompressed, err := Decompress(got)
			if err != nil {
				t.Errorf("Decompress() after Compress() failed: %v", err)
				return
			}

			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("Compress() decompressed data = %v, want %v", decompressed, tt.data)
			}
		})
	}
}

func TestDecompress(t *testing.T) {
	// Create compressed data for testing
	originalData := []byte("hello world")
	compressedData, err := Compress(originalData, gzip.DefaultCompression)
	if err != nil {
		t.Fatalf("Failed to create test compressed data: %v", err)
	}

	tests := []struct {
		name    string
		data    []byte
		want    []byte
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    []byte{},
			want:    []byte{},
			wantErr: false,
		},
		{
			name:    "valid compressed data",
			data:    compressedData,
			want:    originalData,
			wantErr: false,
		},
		{
			name:    "invalid compressed data",
			data:    []byte("not compressed data"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "nil data",
			data:    nil,
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decompress(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decompress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !bytes.Equal(got, tt.want) {
				t.Errorf("Decompress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStreamCompressor_NewWriter(t *testing.T) {
	sc := NewStreamCompressor()
	var buf bytes.Buffer
	writer := sc.NewWriter(&buf)

	if writer == nil {
		t.Fatal("NewWriter() returned nil")
	}

	// Test that writer actually works
	testData := []byte("stream test")
	if _, err := writer.Write(testData); err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Verify compression worked
	decompressed, err := Decompress(buf.Bytes())
	if err != nil {
		t.Fatalf("Decompress() failed: %v", err)
	}
	if !bytes.Equal(decompressed, testData) {
		t.Errorf("Decompressed data = %v, want %v", decompressed, testData)
	}
}

func TestStreamCompressor_NewReader(t *testing.T) {
	// Create compressed data
	testData := []byte("stream reader test")
	compressedData, err := Compress(testData, gzip.DefaultCompression)
	if err != nil {
		t.Fatalf("Failed to compress test data: %v", err)
	}

	sc := NewStreamCompressor()
	reader, err := sc.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		t.Fatalf("NewReader() failed: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}

	if !bytes.Equal(decompressed, testData) {
		t.Errorf("Decompressed data = %v, want %v", decompressed, testData)
	}
}

func TestCompressDecompressRoundTrip(t *testing.T) {
	testCases := [][]byte{
		{},
		{0},
		{1, 2, 3, 4, 5},
		[]byte("Hello, World!"),
		[]byte("The quick brown fox jumps over the lazy dog"),
		bytes.Repeat([]byte("a"), 1000), // larger data
	}

	for i, original := range testCases {
		t.Run(fmt.Sprintf("TestCase_%d", i), func(t *testing.T) {
			compressed, err := Compress(original, gzip.DefaultCompression)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}

			decompressed, err := Decompress(compressed)
			if err != nil {
				t.Fatalf("Decompress failed: %v", err)
			}

			if !bytes.Equal(original, decompressed) {
				t.Errorf("Round-trip failed. Original: %v, Got: %v", original, decompressed)
			}
		})
	}
}

func TestStreamCompressor_RoundTrip(t *testing.T) {
	sc := NewStreamCompressor()
	original := []byte("stream round trip test")

	// Compress using stream
	var compressedBuf bytes.Buffer
	writer := sc.NewWriter(&compressedBuf)
	if _, err := writer.Write(original); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Decompress using stream
	reader, err := sc.NewReader(&compressedBuf)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(original, decompressed) {
		t.Errorf("Stream round-trip failed. Original: %v, Got: %v", original, decompressed)
	}
}
