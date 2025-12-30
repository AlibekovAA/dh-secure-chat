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
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

func TestAuthService_RefreshAccessToken_Success(t *testing.T) {
	svc, mockUserRepo, _, mockRefreshTokenRepo, _, _, _, mockClock := setupAuthService(t)

	refreshToken := "test-refresh-token-123"
	hash := service.HashRefreshToken(refreshToken)
	userID := "user-123"
	username := "testuser"

	storedToken := authdomain.RefreshToken{
		ID:        "token-id",
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: mockClock.Now().Add(24 * time.Hour),
		CreatedAt: mockClock.Now(),
	}

	mockUserRepo.findByIDFunc = func(ctx context.Context, id userdomain.ID) (userdomain.User, error) {
		return userdomain.User{
			ID:           userdomain.ID(userID),
			Username:     username,
			PasswordHash: "hashed",
			CreatedAt:    mockClock.Now(),
		}, nil
	}

	mockTx := &mockRefreshTokenTx{}
	mockTx.findByTokenHashForUpdateFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		if h != hash {
			t.Errorf("expected hash %s, got %s", hash, h)
		}
		return storedToken, nil
	}

	mockTx.deleteByTokenHashFunc = func(ctx context.Context, h string) error {
		return nil
	}

	mockRefreshTokenRepo.txManagerFunc = func() authrepo.RefreshTokenTxManagerInterface {
		return newTestRefreshTokenTxManagerWithFunc(func(ctx context.Context, fn func(context.Context, authrepo.RefreshTokenTx) error) error {
			return fn(ctx, mockTx)
		})
	}

	mockRefreshTokenRepo.createFunc = func(ctx context.Context, token authdomain.RefreshToken) error {
		return nil
	}

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, nil
	}

	result, err := svc.RefreshAccessToken(context.Background(), refreshToken, "127.0.0.1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.AccessToken == "" {
		t.Error("expected access token to be set")
	}

	if result.RefreshToken == "" {
		t.Error("expected refresh token to be set")
	}
}

func TestAuthService_RefreshAccessToken_EmptyToken(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)

	_, err := svc.RefreshAccessToken(context.Background(), "", "127.0.0.1")

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, service.ErrInvalidRefreshToken) {
		t.Errorf("expected ErrInvalidRefreshToken, got %v", err)
	}
}

func TestAuthService_RefreshAccessToken_Expired(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, mockClock := setupAuthService(t)

	refreshToken := "test-refresh-token"
	hash := service.HashRefreshToken(refreshToken)
	userID := "user-123"

	storedToken := authdomain.RefreshToken{
		ID:        "token-id",
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: mockClock.Now().Add(-1 * time.Hour),
		CreatedAt: mockClock.Now().Add(-2 * time.Hour),
	}

	mockTx := &mockRefreshTokenTx{}
	mockTx.findByTokenHashForUpdateFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		return storedToken, nil
	}

	mockTx.deleteByTokenHashFunc = func(ctx context.Context, h string) error {
		return nil
	}

	mockRefreshTokenRepo.txManagerFunc = func() authrepo.RefreshTokenTxManagerInterface {
		return newTestRefreshTokenTxManagerWithFunc(func(ctx context.Context, fn func(context.Context, authrepo.RefreshTokenTx) error) error {
			return fn(ctx, mockTx)
		})
	}

	_, err := svc.RefreshAccessToken(context.Background(), refreshToken, "127.0.0.1")

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, service.ErrRefreshTokenExpired) {
		t.Errorf("expected ErrRefreshTokenExpired, got %v", err)
	}
}

func TestAuthService_RefreshAccessToken_NotFound(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, _ := setupAuthService(t)

	refreshToken := "test-refresh-token"

	mockTx := &mockRefreshTokenTx{}
	mockTx.findByTokenHashForUpdateFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		_ = h
		return authdomain.RefreshToken{}, authrepo.ErrRefreshTokenNotFound
	}

	mockRefreshTokenRepo.txManagerFunc = func() authrepo.RefreshTokenTxManagerInterface {
		return newTestRefreshTokenTxManagerWithFunc(func(ctx context.Context, fn func(context.Context, authrepo.RefreshTokenTx) error) error {
			return fn(ctx, mockTx)
		})
	}

	_, err := svc.RefreshAccessToken(context.Background(), refreshToken, "127.0.0.1")

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, service.ErrInvalidRefreshToken) {
		t.Errorf("expected ErrInvalidRefreshToken, got %v", err)
	}
}

func TestAuthService_RefreshAccessToken_CircuitBreakerOpen(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, _ := setupAuthService(t)

	refreshToken := "test-refresh-token"

	mockTx := &mockRefreshTokenTx{}
	mockTx.findByTokenHashForUpdateFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		return authdomain.RefreshToken{}, commonerrors.ErrCircuitOpen
	}

	mockRefreshTokenRepo.txManagerFunc = func() authrepo.RefreshTokenTxManagerInterface {
		return newTestRefreshTokenTxManagerWithFunc(func(ctx context.Context, fn func(context.Context, authrepo.RefreshTokenTx) error) error {
			return fn(ctx, mockTx)
		})
	}

	_, err := svc.RefreshAccessToken(context.Background(), refreshToken, "127.0.0.1")

	if err == nil {
		t.Fatal("expected error")
	}

	if domainErr, ok := commonerrors.AsDomainError(err); !ok || domainErr.Code() != "SERVICE_UNAVAILABLE" {
		t.Errorf("expected SERVICE_UNAVAILABLE error, got %v", err)
	}
}

func TestAuthService_RefreshAccessToken_UserNotFound(t *testing.T) {
	svc, mockUserRepo, _, mockRefreshTokenRepo, _, _, _, mockClock := setupAuthService(t)

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

	mockTx := &mockRefreshTokenTx{}
	mockTx.findByTokenHashForUpdateFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		return storedToken, nil
	}

	mockUserRepo.findByIDFunc = func(ctx context.Context, id userdomain.ID) (userdomain.User, error) {
		return userdomain.User{}, errors.New("user not found")
	}

	mockRefreshTokenRepo.txManagerFunc = func() authrepo.RefreshTokenTxManagerInterface {
		return newTestRefreshTokenTxManagerWithFunc(func(ctx context.Context, fn func(context.Context, authrepo.RefreshTokenTx) error) error {
			return fn(ctx, mockTx)
		})
	}

	_, err := svc.RefreshAccessToken(context.Background(), refreshToken, "127.0.0.1")

	if err == nil {
		t.Fatal("expected error")
	}
}
