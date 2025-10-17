package middlewares

import (
	"net/http"
)

type Compressor interface {
	CompressResponse(w http.ResponseWriter, r *http.Request) http.ResponseWriter
	DecompressRequest(r *http.Request) error
}

func Compress(compressor Compressor) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := compressor.DecompressRequest(r); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
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
