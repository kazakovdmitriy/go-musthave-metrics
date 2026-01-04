package middlewares

import (
	"net/http"

	"go.uber.org/zap"
)

func RateLimiter(maxConcurrent int, log *zap.Logger) func(next http.Handler) http.Handler {
	semaphore := make(chan struct{}, maxConcurrent)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if maxConcurrent <= 0 {
				log.Warn("Rate limit set to 0 - all requests will be rejected")
				next.ServeHTTP(w, r)
				return
			}

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
				next.ServeHTTP(w, r)
			default:
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			}
		})
	}
}
