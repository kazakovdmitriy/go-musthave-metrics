package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// HybridEncrypt шифрует данные с помощью гибридной схемы
func HybridEncrypt(publicKey *rsa.PublicKey, plaintext []byte) (*EncryptedPayload, error) {
	// 1. Генерируем случайный симметричный ключ
	aesKey := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, err
	}

	// 2. Шифруем данные симметричным ключом (AES-GCM)
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// 3. Шифруем симметричный ключ асимметрично (RSA-OAEP)
	encryptedKey, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		publicKey,
		aesKey,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &EncryptedPayload{
		Data: base64.StdEncoding.EncodeToString(ciphertext),
		Key:  base64.StdEncoding.EncodeToString(encryptedKey),
	}, nil
}

// HybridDecrypt расшифровывает гибридно зашифрованные данные
func HybridDecrypt(privateKey *rsa.PrivateKey, payload *EncryptedPayload) ([]byte, error) {
	if payload.Data == "" || payload.Key == "" {
		return nil, fmt.Errorf("invalid encrypted payload")
	}

	// 1. Расшифровываем симметричный ключ
	encryptedKey, err := base64.StdEncoding.DecodeString(payload.Key)
	if err != nil {
		return nil, err
	}

	aesKey, err := rsa.DecryptOAEP(
		sha256.New(),
		rand.Reader,
		privateKey,
		encryptedKey,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// 2. Расшифровываем данные
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
