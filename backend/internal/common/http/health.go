package http

import (
	"context"
	"net/http"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

type readinessPinger interface {
	Ping(context.Context) error
}

func ReadyHandler(log *logger.Logger, pinger readinessPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pinger.Ping(ctx); err != nil {
			log.Warnf("readiness check failed: %v", err)
			WriteError(w, http.StatusServiceUnavailable, "service not ready")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
