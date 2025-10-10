package middlewares

import (
	"net/http"
	"sync"
)

func TrackActiveRequests(
	activeRequests *sync.WaitGroup,
	shutdownChan chan struct{},
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-shutdownChan:
				http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
				return
			default:
			}

			activeRequests.Add(1)
			defer activeRequests.Done()

			next.ServeHTTP(w, r)
		})
	}
}
