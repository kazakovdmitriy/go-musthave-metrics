package middlewares

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

func RequestLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			duration := time.Since(start)
			log.Debug(
				"got incoming HTTP request",
				zap.String("URI", r.URL.Path),
				zap.String("method", r.Method),
				zap.String("duration", duration.String()),
			)
		})
	}
}

func ResponseLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			sizeKB := float64(ww.BytesWritten()) / 1024.0
			log.Debug(
				"server response",
				zap.Int("status code", ww.Status()),
				zap.String("response size", fmt.Sprintf("%.2f kb", sizeKB)),
			)
		})
	}
}
