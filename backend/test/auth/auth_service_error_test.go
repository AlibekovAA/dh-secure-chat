package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/resilience"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

func TestAuthService_NewAuthService_ClockNil(t *testing.T) {
	mockUserRepo := &mockUserRepo{}
	mockIdentityService := &mockIdentityService{}
	mockRefreshTokenRepo := &mockRefreshTokenRepo{}
	mockRevokedTokenRepo := &mockRevokedTokenRepo{}
	mockHasher := &mockHasher{}
	mockIDGenerator := &mockIDGenerator{}

	log, _ := logger.New("", "test", "info")

	authService := service.NewAuthService(
		service.AuthServiceDeps{
			Repo:             mockUserRepo,
			IdentityService:  mockIdentityService,
			RefreshTokenRepo: mockRefreshTokenRepo,
			RevokedTokenRepo: mockRevokedTokenRepo,
			Hasher:           mockHasher,
			IDGenerator:      mockIDGenerator,
			Clock:            nil,
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

	if authService == nil {
		t.Fatal("expected auth service to be created")
	}
}

func TestAuthService_Register_IssueTokensError(t *testing.T) {
	svc, mockUserRepo, _, mockRefreshTokenRepo, _, mockHasher, _, _ := setupAuthService(t)

	mockHasher.hashFunc = func(p string) (string, error) {
		return "hashed", nil
	}

	mockUserRepo.createFunc = func(ctx context.Context, user userdomain.User) error {
		return nil
	}

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, errors.New("refresh token error")
	}

	_, err := svc.Register(context.Background(), service.RegisterInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthService_Login_IssueTokensError(t *testing.T) {
	svc, mockUserRepo, _, mockRefreshTokenRepo, _, mockHasher, _, mockClock := setupAuthService(t)

	mockUserRepo.findByUsernameFunc = func(ctx context.Context, username string) (userdomain.User, error) {
		return userdomain.User{
			ID:           "user-123",
			Username:     "testuser",
			PasswordHash: "hashed",
			CreatedAt:    mockClock.Now(),
		}, nil
	}

	mockHasher.compareFunc = func(hash string, password string) error {
		return nil
	}

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, errors.New("refresh token error")
	}

	_, err := svc.Login(context.Background(), service.LoginInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthService_RefreshAccessToken_ExpiredDeleteError(t *testing.T) {
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
		return errors.New("delete error")
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

func TestAuthService_RefreshAccessToken_DeleteError(t *testing.T) {
	svc, mockUserRepo, _, mockRefreshTokenRepo, _, _, _, mockClock := setupAuthService(t)

	refreshToken := "test-refresh-token"
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

	mockTx := &mockRefreshTokenTx{}
	mockTx.findByTokenHashForUpdateFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		return storedToken, nil
	}

	mockTx.deleteByTokenHashFunc = func(ctx context.Context, h string) error {
		return errors.New("delete error")
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

func TestAuthService_RefreshAccessToken_IssueTokensError(t *testing.T) {
	svc, mockUserRepo, _, mockRefreshTokenRepo, _, _, _, mockClock := setupAuthService(t)

	refreshToken := "test-refresh-token"
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

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, errors.New("refresh token error")
	}

	_, err := svc.RefreshAccessToken(context.Background(), refreshToken, "127.0.0.1")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthService_RevokeRefreshToken_OtherError(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, _ := setupAuthService(t)

	refreshToken := "test-refresh-token"

	mockRefreshTokenRepo.findByTokenHashFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		return authdomain.RefreshToken{}, errors.New("other database error")
	}

	err := svc.RevokeRefreshToken(context.Background(), refreshToken)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthService_RevokeRefreshToken_DeleteNotFound(t *testing.T) {
	svc, _, _, mockRefreshTokenRepo, _, _, _, mockClock := setupAuthService(t)

	refreshToken := "test-refresh-token"
	hash := service.HashRefreshToken(refreshToken)
	userID := "user-123"

	storedToken := authdomain.RefreshToken{
		ID:        "token-id",
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: mockClock.Now().Add(constants.TestTokenExpiryOffset),
		CreatedAt: mockClock.Now(),
	}

	mockRefreshTokenRepo.findByTokenHashFunc = func(ctx context.Context, h string) (authdomain.RefreshToken, error) {
		return storedToken, nil
	}

	mockRefreshTokenRepo.deleteByTokenHashFunc = func(ctx context.Context, h string) error {
		return authrepo.ErrRefreshTokenNotFound
	}

	err := svc.RevokeRefreshToken(context.Background(), refreshToken)

	if err != nil {
		t.Fatalf("expected no error for not found, got %v", err)
	}
}

func TestAuthService_RevokeAccessToken_OtherError(t *testing.T) {
	svc, _, _, _, mockRevokedTokenRepo, _, _, _ := setupAuthService(t)

	mockRevokedTokenRepo.revokeFunc = func(ctx context.Context, jti string, userID string, expiresAt time.Time) error {
		return errors.New("other database error")
	}

	err := svc.RevokeAccessToken(context.Background(), "jti-123", "user-123")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthService_Register_CreateOtherError(t *testing.T) {
	svc, mockUserRepo, _, _, _, mockHasher, _, _ := setupAuthService(t)

	mockHasher.hashFunc = func(p string) (string, error) {
		return "hashed", nil
	}

	mockUserRepo.createFunc = func(ctx context.Context, user userdomain.User) error {
		return errors.New("other database error")
	}

	_, err := svc.Register(context.Background(), service.RegisterInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if domainErr, ok := commonerrors.AsDomainError(err); !ok || domainErr.Code() != "DB_ERROR" {
		t.Errorf("expected DB_ERROR error, got %v", err)
	}

	if domainErr, ok := commonerrors.AsDomainError(err); ok && domainErr.Message() != "failed to create user" {
		t.Errorf("expected error message 'failed to create user', got %s", domainErr.Message())
	}
}

func setupRefreshTokenRotatorForCoverage(t *testing.T) (*service.RefreshTokenRotator, *mockRefreshTokenRepo, *mockIDGenerator, *clock.MockClock) {
	_ = t
	mockRefreshTokenRepo := &mockRefreshTokenRepo{}
	mockIDGenerator := &mockIDGenerator{}
	mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	log, _ := logger.New("", "test", "info")

	dbCB := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Threshold:  constants.TestCircuitBreakerThreshold,
		Timeout:    constants.TestCircuitBreakerTimeout,
		ResetAfter: constants.TestCircuitBreakerReset,
		Name:       constants.CircuitBreakerDatabaseName,
		Logger:     log,
	})

	rotator := service.NewRefreshTokenRotator(
		mockRefreshTokenRepo,
		dbCB,
		mockIDGenerator,
		constants.DefaultRefreshTokenTTL,
		constants.DefaultMaxRefreshTokensPerUser,
		mockClock,
		log,
	)

	return rotator, mockRefreshTokenRepo, mockIDGenerator, mockClock
}

func TestRefreshTokenRotator_RotateIfNeeded_CountOtherError(t *testing.T) {
	rotator, mockRefreshTokenRepo, _, _ := setupRefreshTokenRotatorForCoverage(t)

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, errors.New("other database error")
	}

	err := rotator.RotateIfNeeded(context.Background(), "user-123")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRefreshTokenRotator_RotateIfNeeded_DeleteOldestError(t *testing.T) {
	rotator, mockRefreshTokenRepo, _, _ := setupRefreshTokenRotatorForCoverage(t)

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return constants.DefaultMaxRefreshTokensPerUser, nil
	}

	mockRefreshTokenRepo.deleteOldestByUserIDFunc = func(ctx context.Context, uid string) error {
		return errors.New("delete error")
	}

	err := rotator.RotateIfNeeded(context.Background(), "user-123")

	if err != nil {
		t.Errorf("expected no error (non-critical), got %v", err)
	}
}

func TestRefreshTokenRotator_IssueRefreshToken_RotateError(t *testing.T) {
	rotator, mockRefreshTokenRepo, _, _ := setupRefreshTokenRotatorForCoverage(t)

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, errors.New("count error")
	}

	user := userdomain.User{
		ID:       "user-123",
		Username: "testuser",
	}

	_, err := rotator.IssueRefreshToken(context.Background(), user)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	validator := service.NewCredentialValidator()

	err := validator.Validate("ab", "password123")
	if err == nil {
		t.Fatal("expected validation error")
	}

	validationErr, ok := service.AsValidationError(err)
	if !ok {
		t.Fatal("expected ValidationError")
	}

	unwrapped := validationErr.Unwrap()
	if unwrapped == nil {
		t.Error("expected unwrapped error")
	}

	if !errors.Is(unwrapped, service.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", unwrapped)
	}
}

func TestAuthService_CloseRefreshTokenCache(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)

	svc.CloseRefreshTokenCache()

	svc.CloseRefreshTokenCache()
}

func TestAuthService_Register_IssueAccessTokenError(t *testing.T) {
	svc, mockUserRepo, _, mockRefreshTokenRepo, _, mockHasher, mockIDGenerator, _ := setupAuthService(t)

	mockHasher.hashFunc = func(p string) (string, error) {
		return "hashed", nil
	}

	callCount := 0
	mockIDGenerator.newIDFunc = func() (string, error) {
		callCount++
		if callCount == 1 {
			return "user-123", nil
		}
		return "", errors.New("id generation failed for token")
	}

	mockUserRepo.createFunc = func(ctx context.Context, user userdomain.User) error {
		return nil
	}

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, nil
	}

	_, err := svc.Register(context.Background(), service.RegisterInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthService_Register_DatabaseError(t *testing.T) {
	svc, mockUserRepo, _, _, _, mockHasher, _, _ := setupAuthService(t)

	mockHasher.hashFunc = func(p string) (string, error) {
		return "hashed", nil
	}

	mockUserRepo.createFunc = func(ctx context.Context, user userdomain.User) error {
		return errors.New("database connection timeout")
	}

	_, err := svc.Register(context.Background(), service.RegisterInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if domainErr, ok := commonerrors.AsDomainError(err); !ok || domainErr.Code() != "DB_ERROR" {
		t.Errorf("expected DB_ERROR error, got %v", err)
	}

	if domainErr, ok := commonerrors.AsDomainError(err); ok && domainErr.Message() != "failed to create user" {
		t.Errorf("expected error message 'failed to create user', got %s", domainErr.Message())
	}
}

func TestAuthService_Register_DatabaseError_UnknownSpecificError(t *testing.T) {
	svc, mockUserRepo, _, _, _, mockHasher, _, _ := setupAuthService(t)

	mockHasher.hashFunc = func(p string) (string, error) {
		return "hashed", nil
	}

	unknownError := errors.New("unknown database error")
	mockUserRepo.createFunc = func(ctx context.Context, user userdomain.User) error {
		return unknownError
	}

	_, err := svc.Register(context.Background(), service.RegisterInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if domainErr, ok := commonerrors.AsDomainError(err); !ok || domainErr.Code() != "DB_ERROR" {
		t.Errorf("expected DB_ERROR error, got %v", err)
	}
}

func TestAuthService_Login_DatabaseError_UnknownSpecificError(t *testing.T) {
	svc, mockUserRepo, _, _, _, _, _, _ := setupAuthService(t)

	unknownError := errors.New("unknown database error")
	mockUserRepo.findByUsernameFunc = func(ctx context.Context, username string) (userdomain.User, error) {
		return userdomain.User{}, unknownError
	}

	_, err := svc.Login(context.Background(), service.LoginInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if domainErr, ok := commonerrors.AsDomainError(err); !ok || domainErr.Code() != "DB_ERROR" {
		t.Errorf("expected DB_ERROR error, got %v", err)
	}
}
