package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

// EncryptedPayload — структура для гибридного шифрования
type EncryptedPayload struct {
	Data string `json:"data"`
	Key  string `json:"key"`
}

// CryptoService — сервис для шифрования/дешифрования
type CryptoService struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
	enabled    bool
	isAgent    bool // true = агент (шифрует), false = сервер (дешифрует)
}

// NewCryptoService создаёт сервис криптографии
func NewCryptoService(keyPath string, isAgent bool) (*CryptoService, error) {
	cs := &CryptoService{
		enabled: false,
		isAgent: isAgent,
	}

	if keyPath == "" {
		return cs, nil
	}

	var err error
	if isAgent {
		cs.publicKey, err = LoadPublicKey(keyPath)
	} else {
		cs.privateKey, err = LoadPrivateKey(keyPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load crypto key: %w", err)
	}

	cs.enabled = true
	return cs, nil
}

// IsEnabled проверяет, включено ли шифрование
func (cs *CryptoService) IsEnabled() bool {
	return cs.enabled
}

// Encrypt шифрует данные (для агента)
func (cs *CryptoService) Encrypt(plaintext []byte) (*EncryptedPayload, error) {
	if !cs.enabled {
		return nil, fmt.Errorf("crypto is disabled")
	}
	if cs.publicKey == nil {
		return nil, fmt.Errorf("public key not loaded")
	}
	return HybridEncrypt(cs.publicKey, plaintext)
}

// Decrypt дешифрует данные (для сервера)
func (cs *CryptoService) Decrypt(payload *EncryptedPayload) ([]byte, error) {
	if !cs.enabled {
		return nil, fmt.Errorf("crypto is disabled")
	}
	if cs.privateKey == nil {
		return nil, fmt.Errorf("private key not loaded")
	}
	return HybridDecrypt(cs.privateKey, payload)
}

// LoadPublicKey загружает публичный ключ из PEM-файла (поддерживает оба формата)
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file %q: %w", path, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %q (is it a valid PEM file?)", path)
	}

	// Поддерживаем оба формата: PKCS#1 и PKCS#8
	var pubKey *rsa.PublicKey
	if block.Type == "RSA PUBLIC KEY" {
		// PKCS#1
		pubKey, err = x509.ParsePKCS1PublicKey(block.Bytes)
	} else if block.Type == "PUBLIC KEY" {
		// PKCS#8 / SubjectPublicKeyInfo
		parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
		}
		var ok bool
		pubKey, ok = parsedKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA (got %T)", parsedKey)
		}
	} else {
		return nil, fmt.Errorf("unsupported public key type %q in %q (expected 'RSA PUBLIC KEY' or 'PUBLIC KEY')",
			block.Type, path)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return pubKey, nil
}

// LoadPrivateKey загружает приватный ключ из PEM-файла (поддерживает PKCS#1 и PKCS#8)
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file %q: %w", path, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		// Показываем первые 100 символов для отладки
		preview := string(data)
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		return nil, fmt.Errorf("failed to decode PEM block from %q. File preview: %q", path, preview)
	}

	// Проверяем, зашифрован ли ключ паролем
	if _, isEncrypted := block.Headers["Proc-Type"]; isEncrypted && strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED") {
		return nil, fmt.Errorf("private key %q is encrypted with a password. Please decrypt it first: openssl rsa -in %s -out %s.decrypted",
			path, path, path)
	}

	var privKey *rsa.PrivateKey
	if block.Type == "RSA PRIVATE KEY" {
		privKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	} else if block.Type == "PRIVATE KEY" {
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS#8 private key: %w", err)
		}
		var ok bool
		privKey, ok = parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA (got %T)", parsedKey)
		}
	} else {
		return nil, fmt.Errorf("unsupported private key type %q in %q (expected 'RSA PRIVATE KEY' or 'PRIVATE KEY')",
			block.Type, path)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return privKey, nil
}
