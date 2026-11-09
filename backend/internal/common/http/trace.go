package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
)

const traceIDHeader = "X-Trace-ID"

var traceIDPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{32}|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$`)

func TraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(traceIDHeader)
		if !traceIDPattern.MatchString(traceID) {
			traceID = generateTraceID()
		}

		w.Header().Set(traceIDHeader, traceID)

		ctx := context.WithValue(r.Context(), constants.TraceIDKey, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate trace ID: " + err.Error())
	}
	return hex.EncodeToString(b)
}
