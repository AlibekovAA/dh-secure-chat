package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

func setupAuthService(t *testing.T) (*service.AuthService, *mockUserRepo, *mockIdentityService, *mockRefreshTokenRepo, *mockRevokedTokenRepo, *mockHasher, *mockIDGenerator, *clock.MockClock) {
	_ = t
	mockUserRepo := &mockUserRepo{}
	mockIdentityService := &mockIdentityService{}
	mockRefreshTokenRepo := &mockRefreshTokenRepo{}
	mockRevokedTokenRepo := &mockRevokedTokenRepo{}
	mockHasher := &mockHasher{}
	mockIDGenerator := &mockIDGenerator{}
	mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))

	log, _ := logger.New("", "test", "info")

	authService := service.NewAuthService(
		service.AuthServiceDeps{
			Repo:             mockUserRepo,
			IdentityService:  mockIdentityService,
			RefreshTokenRepo: mockRefreshTokenRepo,
			RevokedTokenRepo: mockRevokedTokenRepo,
			Hasher:           mockHasher,
			IDGenerator:      mockIDGenerator,
			Clock:            mockClock,
			Log:              log,
		},
		service.AuthServiceConfig{
			JWTSecret:               constants.TestJWTSecret,
			AccessTokenTTL:          constants.TestAccessTokenTTL,
			RefreshTokenTTL:         constants.DefaultRefreshTokenTTL,
			MaxRefreshTokens:        constants.DefaultMaxRefreshTokensPerUser,
			CircuitBreakerThreshold: constants.TestCircuitBreakerThreshold,
			CircuitBreakerTimeout:   constants.TestCircuitBreakerTimeout,
			CircuitBreakerReset:     constants.TestCircuitBreakerReset,
		},
	)

	return authService, mockUserRepo, mockIdentityService, mockRefreshTokenRepo, mockRevokedTokenRepo, mockHasher, mockIDGenerator, mockClock
}

func TestAuthService_Register_Success(t *testing.T) {
	svc, mockUserRepo, mockIdentityService, mockRefreshTokenRepo, _, mockHasher, mockIDGenerator, mockClock := setupAuthService(t)

	userID := "user-123"
	username := "testuser"
	password := "password123"
	hashedPassword := "hashed_password123"

	mockIDGenerator.newIDFunc = func() (string, error) {
		return userID, nil
	}

	mockHasher.hashFunc = func(p string) (string, error) {
		return hashedPassword, nil
	}

	mockUserRepo.createFunc = func(ctx context.Context, user userdomain.User) error {
		if user.Username != username {
			t.Errorf("expected username %s, got %s", username, user.Username)
		}
		if user.PasswordHash != hashedPassword {
			t.Errorf("expected password hash %s, got %s", hashedPassword, user.PasswordHash)
		}
		return nil
	}

	mockIdentityService.createIdentityKeyFunc = func(ctx context.Context, uid string, pubKey []byte) error {
		return nil
	}

	mockRefreshTokenRepo.createFunc = func(ctx context.Context, token authdomain.RefreshToken) error {
		return nil
	}

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		return nil
	}

	result, err := svc.Register(context.Background(), service.RegisterInput{
		Username:       username,
		Password:       password,
		IdentityPubKey: []byte("test-public-key"),
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.AccessToken == "" {
		t.Error("expected access token to be set")
	}

	if result.RefreshToken == "" {
		t.Error("expected refresh token to be set")
	}

	if result.RefreshExpiresAt.Before(mockClock.Now()) {
		t.Error("expected refresh token expiration to be in the future")
	}
}

func TestAuthService_Register_ValidationError(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)

	testCases := []struct {
		name     string
		username string
		password string
	}{
		{"short username", "ab", "password123"},
		{"long username", "a" + string(make([]byte, 33)), "password123"},
		{"short password", "testuser", "pass123"},
		{"long password", "testuser", string(make([]byte, 73))},
		{"invalid username chars", "test@user", "password123"},
		{"username starts with dash", "-testuser", "password123"},
		{"username ends with underscore", "testuser_", "password123"},
		{"password without letter", "testuser", "12345678"},
		{"password without digit", "testuser", "abcdefgh"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), service.RegisterInput{
				Username: tc.username,
				Password: tc.password,
			})

			if err == nil {
				t.Error("expected validation error")
			}

			if domainErr, ok := commonerrors.AsDomainError(err); !ok || domainErr.Code() != "VALIDATION_FAILED" {
				t.Errorf("expected VALIDATION_FAILED error, got %v", err)
			}
		})
	}
}

func TestAuthService_Register_UsernameAlreadyExists(t *testing.T) {
	svc, mockUserRepo, _, _, _, mockHasher, _, _ := setupAuthService(t)

	mockHasher.hashFunc = func(p string) (string, error) {
		return "hashed", nil
	}

	mockUserRepo.createFunc = func(ctx context.Context, user userdomain.User) error {
		return commonerrors.ErrUsernameAlreadyExists
	}

	_, err := svc.Register(context.Background(), service.RegisterInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if domainErr, ok := commonerrors.AsDomainError(err); !ok || domainErr.Code() != "USERNAME_TAKEN" {
		t.Errorf("expected USERNAME_TAKEN error, got %v", err)
	}
}

func TestAuthService_Register_CircuitBreakerOpen(t *testing.T) {
	svc, mockUserRepo, _, _, _, mockHasher, _, _ := setupAuthService(t)

	mockHasher.hashFunc = func(p string) (string, error) {
		return "hashed", nil
	}

	mockUserRepo.createFunc = func(ctx context.Context, user userdomain.User) error {
		return commonerrors.ErrCircuitOpen
	}

	_, err := svc.Register(context.Background(), service.RegisterInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if domainErr, ok := commonerrors.AsDomainError(err); !ok || domainErr.Code() != "SERVICE_UNAVAILABLE" {
		t.Errorf("expected SERVICE_UNAVAILABLE error, got %v", err)
	}
}

func TestAuthService_Register_IdentityKeyError_InvalidKey(t *testing.T) {
	svc, mockUserRepo, mockIdentityService, mockRefreshTokenRepo, _, mockHasher, mockIDGenerator, _ := setupAuthService(t)

	mockIDGenerator.newIDFunc = func() (string, error) {
		return "user-123", nil
	}

	mockHasher.hashFunc = func(p string) (string, error) {
		return "hashed", nil
	}

	mockUserRepo.createFunc = func(ctx context.Context, user userdomain.User) error {
		return nil
	}

	mockIdentityService.createIdentityKeyFunc = func(ctx context.Context, uid string, pubKey []byte) error {
		return commonerrors.ErrInvalidPublicKey
	}

	mockRefreshTokenRepo.createFunc = func(ctx context.Context, token authdomain.RefreshToken) error {
		return nil
	}

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		return nil
	}

	result, err := svc.Register(context.Background(), service.RegisterInput{
		Username:       "testuser",
		Password:       "password123",
		IdentityPubKey: []byte("invalid-key"),
	})

	if err != nil {
		t.Fatalf("expected no error (graceful degradation), got %v", err)
	}

	if result.AccessToken == "" {
		t.Error("expected access token to be set")
	}
}

func TestAuthService_Register_IdentityKeyError_OtherError(t *testing.T) {
	svc, mockUserRepo, mockIdentityService, mockRefreshTokenRepo, _, mockHasher, mockIDGenerator, _ := setupAuthService(t)

	mockIDGenerator.newIDFunc = func() (string, error) {
		return "user-123", nil
	}

	mockHasher.hashFunc = func(p string) (string, error) {
		return "hashed", nil
	}

	mockUserRepo.createFunc = func(ctx context.Context, user userdomain.User) error {
		return nil
	}

	mockIdentityService.createIdentityKeyFunc = func(ctx context.Context, uid string, pubKey []byte) error {
		return errors.New("database error")
	}

	mockRefreshTokenRepo.createFunc = func(ctx context.Context, token authdomain.RefreshToken) error {
		return nil
	}

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		return nil
	}

	result, err := svc.Register(context.Background(), service.RegisterInput{
		Username:       "testuser",
		Password:       "password123",
		IdentityPubKey: []byte("test-key"),
	})

	if err != nil {
		t.Fatalf("expected no error (graceful degradation), got %v", err)
	}

	if result.AccessToken == "" {
		t.Error("expected access token to be set")
	}
}

func TestAuthService_Register_HashError(t *testing.T) {
	svc, _, _, _, _, mockHasher, _, _ := setupAuthService(t)

	mockHasher.hashFunc = func(p string) (string, error) {
		return "", errors.New("hash error")
	}

	_, err := svc.Register(context.Background(), service.RegisterInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthService_Register_IDGenerationError(t *testing.T) {
	svc, _, _, _, _, mockHasher, mockIDGenerator, _ := setupAuthService(t)

	mockHasher.hashFunc = func(p string) (string, error) {
		return "hashed", nil
	}

	mockIDGenerator.newIDFunc = func() (string, error) {
		return "", errors.New("id generation error")
	}

	_, err := svc.Register(context.Background(), service.RegisterInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthService_Register_WithoutIdentityKey(t *testing.T) {
	svc, mockUserRepo, _, mockRefreshTokenRepo, _, mockHasher, mockIDGenerator, mockClock := setupAuthService(t)

	userID := "user-123"
	username := "testuser"
	password := "password123"
	hashedPassword := "hashed_password123"

	mockIDGenerator.newIDFunc = func() (string, error) {
		return userID, nil
	}

	mockHasher.hashFunc = func(p string) (string, error) {
		return hashedPassword, nil
	}

	mockUserRepo.createFunc = func(ctx context.Context, user userdomain.User) error {
		return nil
	}

	mockRefreshTokenRepo.createFunc = func(ctx context.Context, token authdomain.RefreshToken) error {
		return nil
	}

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		return nil
	}

	result, err := svc.Register(context.Background(), service.RegisterInput{
		Username:       username,
		Password:       password,
		IdentityPubKey: nil,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.AccessToken == "" {
		t.Error("expected access token to be set")
	}

	if result.RefreshToken == "" {
		t.Error("expected refresh token to be set")
	}

	if result.RefreshExpiresAt.Before(mockClock.Now()) {
		t.Error("expected refresh token expiration to be in the future")
	}
}
