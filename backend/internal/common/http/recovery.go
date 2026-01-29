package http

import (
	"net/http"
	"runtime/debug"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func RecoveryMiddleware(log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Criticalf("panic recovered: %v\n%s", err, debug.Stack())
					WriteErrorEnvelope(w, http.StatusInternalServerError, CodeUnknown, "internal server error", nil, "")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
