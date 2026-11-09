package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	authhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/config"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

type errorEnvelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func TestAuthHTTP_Register_InvalidJSON(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)
	log, _ := logger.New("", "test", "info")
	cfg := config.AuthConfig{RequestTimeout: 30 * time.Second}
	h := authhttp.NewHandler(svc, cfg, log)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
	var env errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Code != "INVALID_JSON" {
		t.Errorf("expected code INVALID_JSON, got %s", env.Code)
	}
}

func TestAuthHTTP_Register_ValidationError(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)
	log, _ := logger.New("", "test", "info")
	cfg := config.AuthConfig{RequestTimeout: 30 * time.Second}
	h := authhttp.NewHandler(svc, cfg, log)

	body := map[string]string{"username": "ab", "password": "password123"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
	var env errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Code != "VALIDATION_USERNAME_LENGTH" {
		t.Errorf("expected code VALIDATION_USERNAME_LENGTH, got %s", env.Code)
	}
}

func TestAuthHTTP_Login_InvalidJSON(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)
	log, _ := logger.New("", "test", "info")
	cfg := config.AuthConfig{RequestTimeout: 30 * time.Second}
	h := authhttp.NewHandler(svc, cfg, log)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
	var env errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Code != "INVALID_JSON" {
		t.Errorf("expected code INVALID_JSON, got %s", env.Code)
	}
}

func TestAuthHTTP_Login_InvalidCredentials(t *testing.T) {
	svc, mockUserRepo, _, _, _, mockHasher, _, _ := setupAuthService(t)
	mockUserRepo.findByUsernameFunc = func(_ context.Context, username string) (userdomain.User, error) {
		return userdomain.User{
			ID:           "user-123",
			Username:     "testuser",
			PasswordHash: "hashed",
			CreatedAt:    time.Now(),
		}, nil
	}
	mockHasher.compareFunc = func(hash, password string) error {
		return errors.New("mismatch")
	}
	log, _ := logger.New("", "test", "info")
	cfg := config.AuthConfig{RequestTimeout: 30 * time.Second}
	h := authhttp.NewHandler(svc, cfg, log)

	body := map[string]string{"username": "testuser", "password": "wrongpassword1"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
	var env errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected code INVALID_CREDENTIALS, got %s", env.Code)
	}
}

func TestAuthHTTP_Refresh_MissingCookie(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)
	log, _ := logger.New("", "test", "info")
	cfg := config.AuthConfig{RequestTimeout: 30 * time.Second}
	h := authhttp.NewHandler(svc, cfg, log)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
	var env errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Code != "MISSING_REFRESH_TOKEN" {
		t.Errorf("expected code MISSING_REFRESH_TOKEN, got %s", env.Code)
	}
}

func TestAuthHTTP_Register_InvalidIdentityPubKeyEncoding(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)
	log, _ := logger.New("", "test", "info")
	cfg := config.AuthConfig{RequestTimeout: 30 * time.Second}
	h := authhttp.NewHandler(svc, cfg, log)

	body := map[string]interface{}{
		"username":         "testuser",
		"password":         "password123",
		"identity_pub_key": "not-valid-base64!!!",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
	var env errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Code != "INVALID_IDENTITY_PUB_KEY_ENCODING" {
		t.Errorf("expected code INVALID_IDENTITY_PUB_KEY_ENCODING, got %s", env.Code)
	}
}

func TestAuthHTTP_MethodNotAllowed(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)
	log, _ := logger.New("", "test", "info")
	cfg := config.AuthConfig{RequestTimeout: 30 * time.Second}
	h := authhttp.NewHandler(svc, cfg, log)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/register", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
	var env errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf("expected code METHOD_NOT_ALLOWED, got %s", env.Code)
	}
}

func TestAuthHTTP_LogoutAttemptsBothRevocationsAndClearsSecureCookie(t *testing.T) {
	svc, _, _, refreshRepo, revokedRepo, _, _, _ := setupAuthService(t)
	var accessCalled, refreshCalled bool
	revokedRepo.revokeFunc = func(context.Context, string, string, time.Time) error {
		accessCalled = true
		return errors.New("access revoke failed")
	}
	refreshRepo.findByTokenHashFunc = func(context.Context, string) (authdomain.RefreshToken, error) {
		return authdomain.RefreshToken{UserID: "user-123"}, nil
	}
	refreshRepo.deleteByTokenHashFunc = func(context.Context, string) error {
		refreshCalled = true
		return errors.New("refresh revoke failed")
	}
	now := time.Now()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-123", "usr": "testuser", "jti": "token-id", "iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
	}).SignedString([]byte(constants.TestJWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	log, err := logger.New(t.TempDir(), "test", "error")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	handler := authhttp.NewHandler(svc, config.AuthConfig{RequestTimeout: time.Second, SecureCookies: true}, log)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
	if !accessCalled || !refreshCalled {
		t.Fatalf("revocations called: access=%v refresh=%v", accessCalled, refreshCalled)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "refresh_token" || cookies[0].MaxAge != -1 || !cookies[0].Secure {
		t.Fatalf("refresh cookie was not securely cleared: %#v", cookies)
	}
}

func TestAuthHTTP_RegisterSetsSecureRefreshCookie(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)
	log, err := logger.New(t.TempDir(), "test", "error")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	handler := authhttp.NewHandler(svc, config.AuthConfig{RequestTimeout: time.Second, SecureCookies: true}, log)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"testuser","password":"password123"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("refresh cookie attributes are unsafe: %#v", cookies)
	}
}
