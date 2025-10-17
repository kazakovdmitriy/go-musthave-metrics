package compressor

import (
	"net/http"
)

type Compressor interface {
	CompressResponse(w http.ResponseWriter, r *http.Request) http.ResponseWriter
	DecompressRequest(r *http.Request) error
}
