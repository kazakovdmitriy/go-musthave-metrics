package signerservice

import (
	"crypto/sha256"
	"fmt"
)

type SHA256Signer struct {
	key string
}

func NewSHA256Signer(key string) *SHA256Signer {
	return &SHA256Signer{key: key}
}

func (s *SHA256Signer) Sign(data []byte) string {
	h := sha256.New()
	h.Write([]byte(s.key))
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (s *SHA256Signer) Verify(data []byte, expectedHash string) bool {
	return s.Sign(data) == expectedHash
}
