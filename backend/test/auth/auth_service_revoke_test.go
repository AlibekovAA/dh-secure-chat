package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

func TestAuthService_RevokeRefreshToken_Success(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, mockClock := setupAuthService(t)

	refreshToken := "test-refresh-token"
	hash := service.HashRefreshToken(refreshToken)
	userID := "user-123"

	storedToken := authdomain.RefreshToken{
		ID:        "token-id",
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: mockClock.Now().Add(24 * time.Hour),
		CreatedAt: mockClock.Now(),
	}

	mockRefreshTokenRepo.findByTokenHashFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		if h != hash {
			t.Errorf("expected hash %s, got %s", hash, h)
		}
		return storedToken, nil
	}

	mockRefreshTokenRepo.deleteByTokenHashFunc = func(ctx context.Context, h string) error {
		return nil
	}

	err := svc.RevokeRefreshToken(context.Background(), refreshToken)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAuthService_RevokeRefreshToken_EmptyToken(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)

	err := svc.RevokeRefreshToken(context.Background(), "")

	if err != nil {
		t.Fatalf("expected no error for empty token, got %v", err)
	}
}

func TestAuthService_RevokeRefreshToken_NotFound(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, _ := setupAuthService(t)

	refreshToken := "test-refresh-token"

	mockRefreshTokenRepo.findByTokenHashFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		_ = h
		return authdomain.RefreshToken{}, authrepo.ErrRefreshTokenNotFound
	}

	err := svc.RevokeRefreshToken(context.Background(), refreshToken)

	if err != nil {
		t.Fatalf("expected no error for not found token, got %v", err)
	}
}

func TestAuthService_RevokeRefreshToken_CircuitBreakerOpen(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, _ := setupAuthService(t)

	refreshToken := "test-refresh-token"

	mockRefreshTokenRepo.findByTokenHashFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		_ = h
		return authdomain.RefreshToken{}, commonerrors.ErrCircuitOpen
	}

	err := svc.RevokeRefreshToken(context.Background(), refreshToken)

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, commonerrors.ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestAuthService_RevokeRefreshToken_DeleteError(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, mockClock := setupAuthService(t)

	refreshToken := "test-refresh-token"
	hash := service.HashRefreshToken(refreshToken)
	userID := "user-123"

	storedToken := authdomain.RefreshToken{
		ID:        "token-id",
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: mockClock.Now().Add(24 * time.Hour),
		CreatedAt: mockClock.Now(),
	}

	mockRefreshTokenRepo.findByTokenHashFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		return storedToken, nil
	}

	mockRefreshTokenRepo.deleteByTokenHashFunc = func(ctx context.Context, h string) error {
		return errors.New("delete error")
	}

	err := svc.RevokeRefreshToken(context.Background(), refreshToken)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthService_RevokeAccessToken_Success(t *testing.T) {
	svc, _, _, _, mockRevokedTokenRepo, _, _, _ := setupAuthService(t)

	jti := "jti-123"
	userID := "user-123"

	mockRevokedTokenRepo.revokeFunc = func(ctx context.Context, j string, uid string, expiresAt time.Time) error {
		if j != jti {
			t.Errorf("expected jti %s, got %s", jti, j)
		}
		if uid != userID {
			t.Errorf("expected userID %s, got %s", userID, uid)
		}
		_ = expiresAt
		return nil
	}

	err := svc.RevokeAccessToken(context.Background(), jti, userID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAuthService_RevokeAccessToken_EmptyJTI(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)

	err := svc.RevokeAccessToken(context.Background(), "", "user-123")

	if err != nil {
		t.Fatalf("expected no error for empty jti, got %v", err)
	}
}

func TestAuthService_RevokeAccessToken_CircuitBreakerOpen(t *testing.T) {
	svc, _, _, _, mockRevokedTokenRepo, _, _, _ := setupAuthService(t)

	mockRevokedTokenRepo.revokeFunc = func(ctx context.Context, jti string, userID string, expiresAt time.Time) error {
		return commonerrors.ErrCircuitOpen
	}

	err := svc.RevokeAccessToken(context.Background(), "jti-123", "user-123")

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, commonerrors.ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestAuthService_ParseTokenForRevoke_InvalidToken(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)

	_, err := svc.ParseTokenForRevoke(context.Background(), "invalid-token")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthService_ParseTokenForRevoke_EmptyToken(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)

	_, err := svc.ParseTokenForRevoke(context.Background(), "")

	if err == nil {
		t.Fatal("expected error")
	}
}
