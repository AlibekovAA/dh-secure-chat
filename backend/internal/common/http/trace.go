package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type contextKey string

const traceIDKey contextKey = "trace_id"

const traceIDHeader = "X-Trace-ID"

func TraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(traceIDHeader)
		if traceID == "" {
			traceID = generateTraceID()
		}

		w.Header().Set(traceIDHeader, traceID)

		ctx := context.WithValue(r.Context(), traceIDKey, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
