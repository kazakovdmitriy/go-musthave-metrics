package signerservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSHA256Signer_Sign(t *testing.T) {
	key := "test_key"
	signer := NewSHA256Signer(key)

	t.Run("signs empty data", func(t *testing.T) {
		// SHA256(key) = SHA256("test_key") = 92488e1e3eeecdf99f3ed2ce59233efb4b4fb612d5655c0ce9ea52b5a502e655
		expectedHash := "92488e1e3eeecdf99f3ed2ce59233efb4b4fb612d5655c0ce9ea52b5a502e655"
		data := []byte("")

		hash := signer.Sign(data)

		assert.Equal(t, expectedHash, hash)
	})

	t.Run("signs non-empty data", func(t *testing.T) {
		// SHA256(key + data) = SHA256("test_keyHello, World!") = 0774f2b9a63c4abc12a829b3fdb46c24c13b9da4e81d42b67df9f60447914a10
		expectedHash := "0774f2b9a63c4abc12a829b3fdb46c24c13b9da4e81d42b67df9f60447914a10"
		data := []byte("Hello, World!")

		hash := signer.Sign(data)

		assert.Equal(t, expectedHash, hash)
	})

	t.Run("signs different data produces different hashes", func(t *testing.T) {
		data1 := []byte("data one")
		data2 := []byte("data two")

		hash1 := signer.Sign(data1)
		hash2 := signer.Sign(data2)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("signs same data produces same hash", func(t *testing.T) {
		data := []byte("consistent data")

		hash1 := signer.Sign(data)
		hash2 := signer.Sign(data)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("signs with different keys produces different hashes", func(t *testing.T) {
		data := []byte("same data")
		signer1 := NewSHA256Signer("key1")
		signer2 := NewSHA256Signer("key2")

		hash1 := signer1.Sign(data)
		hash2 := signer2.Sign(data)

		assert.NotEqual(t, hash1, hash2)
	})
}

func TestSHA256Signer_Verify(t *testing.T) {
	key := "verify_key"
	signer := NewSHA256Signer(key)

	t.Run("valid hash returns true", func(t *testing.T) {
		data := []byte("test data for verification")
		correctHash := signer.Sign(data)

		isValid := signer.Verify(data, correctHash)

		assert.True(t, isValid)
	})

	t.Run("invalid hash returns false", func(t *testing.T) {
		data := []byte("test data for verification")
		// Явно указываем неправильный хеш
		incorrectHash := "aabbccddeeff0011223344556677889900112233445566778899001122334455"

		isValid := signer.Verify(data, incorrectHash)

		assert.False(t, isValid)
	})

	t.Run("hash from different data returns false", func(t *testing.T) {
		data1 := []byte("original data")
		data2 := []byte("different data")
		hashFromData2 := signer.Sign(data2)

		// Проверяем, что хеш от data2 не подходит для data1
		isValid := signer.Verify(data1, hashFromData2)

		assert.False(t, isValid)
	})

	t.Run("hash from same data with different signer key returns false", func(t *testing.T) {
		data := []byte("shared data")
		signer1 := NewSHA256Signer("key1")
		signer2 := NewSHA256Signer("key2")

		hashFromSigner1 := signer1.Sign(data)

		// Проверяем, что хеш, созданный signer1, не проходит проверку у signer2
		isValid := signer2.Verify(data, hashFromSigner1)

		assert.False(t, isValid)
	})
}

func TestNewSHA256Signer(t *testing.T) {
	key := "new_signer_key"
	signer := NewSHA256Signer(key)

	require.NotNil(t, signer)
	assert.Equal(t, key, signer.key)
}
