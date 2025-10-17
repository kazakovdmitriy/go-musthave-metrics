package compressor

import (
	"net/http"

	"go.uber.org/zap"
)

func Compress(compressor Compressor, log *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := compressor.DecompressRequest(r); err != nil {
				msg := http.StatusText(http.StatusInternalServerError)
				http.Error(w, msg, http.StatusInternalServerError)
				log.Error("Decompression error", zap.Error(err))
				return
			}

			compressWriter := compressor.CompressResponse(w, r)

			if cw, ok := compressWriter.(interface{ Close() error }); ok {
				defer cw.Close()
				w = compressWriter
			}

			next.ServeHTTP(w, r)
		})
	}
}
