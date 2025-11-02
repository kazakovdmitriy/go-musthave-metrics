package signer

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
)

func HashValidationMiddleware(signer Signer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if signer == nil {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusBadRequest)
			}
			defer r.Body.Close()

			// Восстанавливаем body для следующего обработчика
			r.Body = io.NopCloser(bytes.NewBuffer(body))

			givenHash := r.Header.Get("HashSHA256")
			if givenHash == "" {
				http.Error(w, "missing HashSHA256 header", http.StatusBadRequest)
				return
			}

			if !signer.Verify(body, givenHash) {
				http.Error(w, "invalid hash", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func HashResponseMiddleware(signer Signer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if signer == nil {
				next.ServeHTTP(w, r)
				return
			}

			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)

			body := rec.Body.Bytes()
			hash := signer.Sign(body)

			for k, vs := range rec.Header() {
				for _, v := range vs {
					w.Header().Add(k, v)
				}
			}
			w.Header().Set("HashSHA256", hash)
			w.WriteHeader(rec.Code)
			w.Write(body)
		})
	}
}
