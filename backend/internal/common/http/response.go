package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

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
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain a single JSON value")
		}
		return err
	}
	return nil
}

func WriteDecodeJSONError(w http.ResponseWriter, err error) {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		WriteErrorEnvelope(w, http.StatusRequestEntityTooLarge, CodeRequestBodyTooLarge, "request body too large", nil, "")
		return
	}
	WriteErrorEnvelope(w, http.StatusBadRequest, CodeInvalidJSON, "invalid json", nil, "")
}

func GetClientIP(r *http.Request) string {
	remoteIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		remoteIP = host
	}
	parsedRemoteIP := net.ParseIP(remoteIP)
	if parsedRemoteIP == nil || (!parsedRemoteIP.IsPrivate() && !parsedRemoteIP.IsLoopback()) {
		return remoteIP
	}

	ip := r.Header.Get("X-Real-IP")
	if net.ParseIP(ip) == nil {
		ip = r.Header.Get("X-Forwarded-For")
		if idx := strings.Index(ip, ","); idx != -1 {
			ip = strings.TrimSpace(ip[:idx])
		}
	}
	if net.ParseIP(ip) == nil {
		return remoteIP
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
