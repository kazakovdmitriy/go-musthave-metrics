package logger

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

func Initialise(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	Log = zl
	return nil
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		Log.Info(
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
		Log.Info(
			"server response",
			zap.Int("status code", ww.Status()),
			zap.String("response size", fmt.Sprintf("%.2f kb", sizeKB)),
		)
	})
}
