package signer

type Signer interface {
	Sign(data []byte) string
	Verify(data []byte, expectedHash string) bool
}
