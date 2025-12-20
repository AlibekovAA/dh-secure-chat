package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}

func RequireMethod(method string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			next(w, r)
		}
	}
}

func WithTimeout(timeout time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next(w, r.WithContext(ctx))
		}
	}
}

func HandleError(w http.ResponseWriter, log *logger.Logger, err error, defaultMsg string) {
	if err == nil {
		return
	}
	log.Errorf("handler error: %v", err)
	WriteError(w, http.StatusInternalServerError, defaultMsg)
}
