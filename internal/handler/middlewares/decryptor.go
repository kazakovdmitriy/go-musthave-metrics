package middlewares

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/utils/crypto"
	"go.uber.org/zap"
	"io"
	"net/http"
)

// Decryptor — структура для хранения приватного ключа
type Decryptor struct {
	privateKey *rsa.PrivateKey
	log        *zap.Logger
	enabled    bool
}

func NewDecryptor(keyPath string, log *zap.Logger) (*Decryptor, error) {
	d := &Decryptor{
		log:     log,
		enabled: false,
	}

	if keyPath == "" {
		log.Info("crypto: decryption disabled (no key path)")
		return d, nil
	}

	privateKey, err := crypto.LoadPrivateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("crypto: unable to load private key: %w", err)
	}

	d.privateKey = privateKey
	d.enabled = true
	log.Info("crypto: decryption enabled", zap.String("keyPath", keyPath))

	return d, nil
}

func (d *Decryptor) Middleware(next http.Handler) http.Handler {
	if !d.enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil || r.ContentLength == 0 {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			d.log.Error("crypto: failed to read request body", zap.Error(err))
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Проверяем, зашифровано ли тело (ищем структуру гибридного шифрования)
		encrypted := crypto.EncryptedPayload{}
		if err := json.Unmarshal(body, &encrypted); err != nil {
			// Не удалось распарсить как зашифрованное — передаём как есть
			// (возможно, это обычный запрос без шифрования)
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
			next.ServeHTTP(w, r)
			return
		}

		// Проверяем наличие обязательных полей шифрования
		if encrypted.Data == "" || encrypted.Key == "" {
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
			next.ServeHTTP(w, r)
			return
		}

		// Дешифруем данные
		plaintext, err := crypto.HybridDecrypt(d.privateKey, &encrypted)
		if err != nil {
			d.log.Error("crypto: decryption failed", zap.Error(err))
			http.Error(w, "decryption failed", http.StatusBadRequest)
			return
		}

		// Заменяем тело запроса на расшифрованные данные
		r.Body = io.NopCloser(bytes.NewReader(plaintext))
		r.ContentLength = int64(len(plaintext))

		d.log.Debug("crypto: request decrypted successfully")

		// Передаём управление следующему обработчику
		next.ServeHTTP(w, r)
	})
}
