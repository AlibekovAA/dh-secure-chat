package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type ErrorEnvelope struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	TraceID string         `json:"trace_id,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteErrorEnvelope(w, status, CodeUnknown, message, nil, "")
}

func WriteErrorEnvelope(w http.ResponseWriter, status int, code, message string, details map[string]any, traceID string) {
	env := ErrorEnvelope{Code: code, Message: message}
	if len(details) > 0 {
		env.Details = details
	}
	if traceID != "" {
		env.TraceID = traceID
	}
	WriteJSON(w, status, env)
}

func DecodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func GetClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
		if idx := strings.Index(ip, ","); idx != -1 {
			ip = strings.TrimSpace(ip[:idx])
		}
	}
	if ip == "" {
		ip = r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
	}
	return ip
}

func RequireMethod(method string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				WriteErrorEnvelope(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed", nil, "")
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
