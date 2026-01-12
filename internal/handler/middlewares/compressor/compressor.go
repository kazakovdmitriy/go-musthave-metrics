package compressor

import (
	"errors"
	"net/http"

	"go.uber.org/zap"
)

func Compress(compressor Compressor, log *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := compressor.DecompressRequest(r); err != nil {
				log.Error("Decompression error", zap.Error(err))
				http.Error(w, "Bad Request: invalid compressed body", http.StatusInternalServerError)
				return
			}

			responseWriter := compressor.CompressResponse(w, r)

			defer func() {
				if cw, ok := responseWriter.(interface{ Close() error }); ok {
					if err := cw.Close(); err != nil && !errors.Is(err, http.ErrAbortHandler) {
						log.Error("Failed to close compress writer", zap.Error(err))
					}
				}
			}()

			next.ServeHTTP(responseWriter, r)
		})
	}
}
