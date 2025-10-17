package middlewares

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/logger"
	"go.uber.org/zap"
)

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		logger.Log.Info(
			"got incoming HTTP request",
			zap.String("URI", r.URL.Path),
			zap.String("method", r.Method),
			zap.String("duration", duration.String()),
		)
	})
}

func ResponseLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// wr := &responseWriter{ResponseWriter: w}
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		sizeKB := float64(ww.BytesWritten()) / 1024.0
		logger.Log.Info(
			"server response",
			zap.Int("status code", ww.Status()),
			zap.String("response size", fmt.Sprintf("%.2f kb", sizeKB)),
		)
	})
}
