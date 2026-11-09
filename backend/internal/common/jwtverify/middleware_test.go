package jwtverify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/golang-jwt/jwt/v5"
)

func signedToken(t *testing.T, secret []byte, claims jwt.MapClaims, method jwt.SigningMethod) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

type revokedChecker bool

func (checker revokedChecker) IsRevoked(context.Context, string) (bool, error) {
	return bool(checker), nil
}

func TestMiddlewareAcceptsValidAndRejectsRevokedToken(t *testing.T) {
	secret := []byte("test-secret-that-is-long-enough-12345")
	now := time.Now()
	token := signedToken(t, secret, jwt.MapClaims{"sub": "user", "usr": "name", "jti": "id", "iat": now.Unix(), "exp": now.Add(time.Hour).Unix()}, jwt.SigningMethodHS256)
	log, err := logger.New(t.TempDir(), "test", "ERROR")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	for name, test := range map[string]struct {
		revoked bool
		status  int
	}{
		"valid":   {status: http.StatusNoContent},
		"revoked": {revoked: true, status: http.StatusUnauthorized},
	} {
		t.Run(name, func(t *testing.T) {
			handler := Middleware(string(secret), log, revokedChecker(test.revoked))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("got %d", recorder.Code)
			}
		})
	}
}

func TestParseTokenRequiresSecurityClaims(t *testing.T) {
	secret := []byte("test-secret-that-is-long-enough-12345")
	now := time.Now()
	base := jwt.MapClaims{"sub": "user", "usr": "name", "jti": "id", "iat": now.Unix(), "exp": now.Add(time.Hour).Unix()}

	for _, claim := range []string{"jti", "iat", "exp"} {
		t.Run("missing_"+claim, func(t *testing.T) {
			claims := jwt.MapClaims{}
			for key, value := range base {
				if key != claim {
					claims[key] = value
				}
			}
			if _, err := ParseToken(signedToken(t, secret, claims, jwt.SigningMethodHS256), secret); err == nil {
				t.Fatalf("expected token without %s to fail", claim)
			}
		})
	}

	if _, err := ParseToken(signedToken(t, secret, base, jwt.SigningMethodHS256), secret); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	expired := jwt.MapClaims{"sub": "user", "usr": "name", "jti": "id", "iat": now.Add(-2 * time.Hour).Unix(), "exp": now.Add(-time.Hour).Unix()}
	if _, err := ParseToken(signedToken(t, secret, expired, jwt.SigningMethodHS256), secret); err == nil {
		t.Fatal("expected expired token to fail")
	}
	if _, err := ParseToken(signedToken(t, secret, base, jwt.SigningMethodHS384), secret); err == nil {
		t.Fatal("expected non-HS256 token to fail")
	}
}
