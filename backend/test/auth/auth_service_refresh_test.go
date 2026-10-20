package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
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
		ExpiresAt: mockClock.Now().Add(constants.TestTokenExpiryOffset),
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

	mockUser := userdomain.User{
		ID:           userdomain.ID(userID),
		Username:     username,
		PasswordHash: "hashed",
		CreatedAt:    mockClock.Now(),
	}

	mockTx := &mockRefreshTokenTx{}
	mockTx.findByTokenHashWithUserForUpdateFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, userdomain.User, error) {
		if h != hash {
			t.Errorf("expected hash %s, got %s", hash, h)
		}
		return storedToken, mockUser, nil
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

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		return nil
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

	mockUser := userdomain.User{
		ID:           userdomain.ID(userID),
		Username:     "testuser",
		PasswordHash: "hashed",
		CreatedAt:    mockClock.Now(),
	}

	mockTx := &mockRefreshTokenTx{}
	mockTx.findByTokenHashWithUserForUpdateFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, userdomain.User, error) {
		return storedToken, mockUser, nil
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
	mockTx.findByTokenHashWithUserForUpdateFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, userdomain.User, error) {
		_ = h
		return authdomain.RefreshToken{}, userdomain.User{}, authrepo.ErrRefreshTokenNotFound
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

func TestAuthService_RefreshAccessToken_DeleteNotFound(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, mockClock := setupAuthService(t)

	refreshToken := "test-refresh-token"
	storedToken := authdomain.RefreshToken{
		ID:        "token-id",
		TokenHash: service.HashRefreshToken(refreshToken),
		UserID:    "user-123",
		ExpiresAt: mockClock.Now().Add(constants.TestTokenExpiryOffset),
		CreatedAt: mockClock.Now(),
	}
	user := userdomain.User{
		ID:        userdomain.ID(storedToken.UserID),
		Username:  "testuser",
		CreatedAt: mockClock.Now(),
	}

	mockTx := &mockRefreshTokenTx{}
	mockTx.findByTokenHashWithUserForUpdateFunc = func(ctx context.Context, hash string) (authdomain.RefreshToken, userdomain.User, error) {
		return storedToken, user, nil
	}
	mockTx.deleteByTokenHashFunc = func(ctx context.Context, hash string) error {
		return authrepo.ErrRefreshTokenNotFound
	}
	mockRefreshTokenRepo.txManagerFunc = func() authrepo.RefreshTokenTxManagerInterface {
		return newTestRefreshTokenTxManagerWithFunc(func(ctx context.Context, fn func(context.Context, authrepo.RefreshTokenTx) error) error {
			return fn(ctx, mockTx)
		})
	}
	mockRefreshTokenRepo.createFunc = func(ctx context.Context, token authdomain.RefreshToken) error {
		t.Fatal("must not issue a new refresh token after failed consumption")
		return nil
	}

	_, err := svc.RefreshAccessToken(context.Background(), refreshToken, "127.0.0.1")

	if !errors.Is(err, service.ErrInvalidRefreshToken) {
		t.Errorf("expected ErrInvalidRefreshToken, got %v", err)
	}
}

func TestAuthService_RefreshAccessToken_RejectsRepeatedUse(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, mockClock := setupAuthService(t)

	refreshToken := "test-refresh-token"
	storedToken := authdomain.RefreshToken{
		ID:        "token-id",
		TokenHash: service.HashRefreshToken(refreshToken),
		UserID:    "user-123",
		ExpiresAt: mockClock.Now().Add(constants.TestTokenExpiryOffset),
		CreatedAt: mockClock.Now(),
	}
	user := userdomain.User{
		ID:        userdomain.ID(storedToken.UserID),
		Username:  "testuser",
		CreatedAt: mockClock.Now(),
	}

	available := true
	findCalls := 0
	deleteCalls := 0
	mockTx := &mockRefreshTokenTx{}
	mockTx.findByTokenHashWithUserForUpdateFunc = func(ctx context.Context, hash string) (authdomain.RefreshToken, userdomain.User, error) {
		findCalls++
		if !available {
			return authdomain.RefreshToken{}, userdomain.User{}, authrepo.ErrRefreshTokenNotFound
		}
		return storedToken, user, nil
	}
	mockTx.deleteByTokenHashFunc = func(ctx context.Context, hash string) error {
		deleteCalls++
		if !available {
			return authrepo.ErrRefreshTokenNotFound
		}
		available = false
		return nil
	}
	mockRefreshTokenRepo.txManagerFunc = func() authrepo.RefreshTokenTxManagerInterface {
		return newTestRefreshTokenTxManagerWithFunc(func(ctx context.Context, fn func(context.Context, authrepo.RefreshTokenTx) error) error {
			return fn(ctx, mockTx)
		})
	}

	if _, err := svc.RefreshAccessToken(context.Background(), refreshToken, "127.0.0.1"); err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}

	_, err := svc.RefreshAccessToken(context.Background(), refreshToken, "127.0.0.1")
	if !errors.Is(err, service.ErrInvalidRefreshToken) {
		t.Errorf("expected ErrInvalidRefreshToken on repeated use, got %v", err)
	}
	if findCalls != 2 {
		t.Errorf("expected database lookup on every use, got %d lookups", findCalls)
	}
	if deleteCalls != 1 {
		t.Errorf("expected one successful token deletion, got %d delete attempts", deleteCalls)
	}
}

func TestAuthService_RefreshAccessToken_CircuitBreakerOpen(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, _ := setupAuthService(t)

	refreshToken := "test-refresh-token"

	mockTx := &mockRefreshTokenTx{}
	mockTx.findByTokenHashWithUserForUpdateFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, userdomain.User, error) {
		return authdomain.RefreshToken{}, userdomain.User{}, commonerrors.ErrCircuitOpen
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
