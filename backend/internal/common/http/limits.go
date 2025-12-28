package http

import (
	"io"
	"net/http"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
)

const (
	DefaultMaxRequestSize = constants.DefaultMaxRequestSize
)

type maxBytesReader struct {
	reader io.ReadCloser
	limit  int64
	read   int64
}

func (r *maxBytesReader) Read(p []byte) (n int, err error) {
	if r.read >= r.limit {
		return 0, http.ErrBodyReadAfterClose
	}
	n, err = r.reader.Read(p)
	r.read += int64(n)
	if r.read > r.limit {
		return n, http.ErrBodyReadAfterClose
	}
	return n, err
}

func (r *maxBytesReader) Close() error {
	return r.reader.Close()
}

func MaxRequestSizeMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxRequestSize
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}

			r.Body = &maxBytesReader{
				reader: r.Body,
				limit:  maxBytes,
			}

			next.ServeHTTP(w, r)
		})
	}
}
