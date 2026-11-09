package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func decodeHandler(w http.ResponseWriter, r *http.Request) {
	var value map[string]string
	if err := DecodeJSON(r, &value); err != nil {
		WriteDecodeJSONError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, value)
}

func TestRequestBodyLimitAndSingleJSONValue(t *testing.T) {
	handler := MaxRequestSizeMiddleware(13)(http.HandlerFunc(decodeHandler))
	exact := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":"12345"}`))
	exactRecorder := httptest.NewRecorder()
	handler.ServeHTTP(exactRecorder, exact)
	if exactRecorder.Code != http.StatusOK {
		t.Fatalf("exact limit returned %d", exactRecorder.Code)
	}

	over := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":"123456"}`))
	over.ContentLength = -1
	overRecorder := httptest.NewRecorder()
	handler.ServeHTTP(overRecorder, over)
	if overRecorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("over limit returned %d", overRecorder.Code)
	}

	trailing := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{} {}`))
	trailingRecorder := httptest.NewRecorder()
	MaxRequestSizeMiddleware(100)(http.HandlerFunc(decodeHandler)).ServeHTTP(trailingRecorder, trailing)
	if trailingRecorder.Code != http.StatusBadRequest {
		t.Fatalf("trailing JSON returned %d", trailingRecorder.Code)
	}

	unknown := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":"ok","unknown":"value"}`))
	unknownRecorder := httptest.NewRecorder()
	type strictRequest struct {
		A string `json:"a"`
	}
	strictHandler := MaxRequestSizeMiddleware(100)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var value strictRequest
		if err := DecodeJSON(r, &value); err != nil {
			WriteDecodeJSONError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	strictHandler.ServeHTTP(unknownRecorder, unknown)
	if unknownRecorder.Code != http.StatusBadRequest {
		t.Fatalf("unknown field returned %d", unknownRecorder.Code)
	}
}

func TestTraceIDValidation(t *testing.T) {
	handler := TraceIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	valid := "0123456789abcdef0123456789abcdef"
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(traceIDHeader, valid)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Header().Get(traceIDHeader) != valid {
		t.Fatal("valid trace ID was not preserved")
	}

	request = httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(traceIDHeader, strings.Repeat("x", 1000))
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	generated := recorder.Header().Get(traceIDHeader)
	if generated == strings.Repeat("x", 1000) || !traceIDPattern.MatchString(generated) {
		t.Fatalf("invalid generated trace ID: %q", generated)
	}
}

func TestGetClientIPTrustsForwardingHeadersOnlyFromPrivateProxy(t *testing.T) {
	direct := httptest.NewRequest(http.MethodGet, "/", nil)
	direct.RemoteAddr = "203.0.113.10:1234"
	direct.Header.Set("X-Real-IP", "198.51.100.20")
	if got := GetClientIP(direct); got != "203.0.113.10" {
		t.Fatalf("trusted spoofed header: %s", got)
	}

	proxied := httptest.NewRequest(http.MethodGet, "/", nil)
	proxied.RemoteAddr = "172.18.0.2:1234"
	proxied.Header.Set("X-Real-IP", "198.51.100.20")
	if got := GetClientIP(proxied); got != "198.51.100.20" {
		t.Fatalf("ignored trusted proxy header: %s", got)
	}
}

type readinessStub struct{ err error }

func (stub readinessStub) Ping(context.Context) error { return stub.err }

func TestReadyHandler(t *testing.T) {
	log, err := logger.New(t.TempDir(), "test", "ERROR")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	for name, test := range map[string]struct {
		err    error
		status int
	}{
		"ready":       {status: http.StatusOK},
		"unavailable": {err: errors.New("db down"), status: http.StatusServiceUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ReadyHandler(log, readinessStub{err: test.err}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))
			if recorder.Code != test.status {
				t.Fatalf("got %d", recorder.Code)
			}
		})
	}
}
