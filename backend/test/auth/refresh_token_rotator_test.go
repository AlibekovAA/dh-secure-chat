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
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/resilience"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

func setupRefreshTokenRotator(t *testing.T) (*service.RefreshTokenRotator, *mockRefreshTokenRepo, *mockIDGenerator, *clock.MockClock) {
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

func TestRefreshTokenRotator_IssueRefreshToken_Success(t *testing.T) {
	rotator, mockRefreshTokenRepo, mockIDGenerator, mockClock := setupRefreshTokenRotator(t)

	userID := "user-123"
	tokenID := "token-id-123"

	mockIDGenerator.newIDFunc = func() (string, error) {
		return tokenID, nil
	}

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		return nil
	}

	mockRefreshTokenRepo.createFunc = func(ctx context.Context, token authdomain.RefreshToken) error {
		if token.UserID != userID {
			t.Errorf("expected userID %s, got %s", userID, token.UserID)
		}
		if token.ID != tokenID {
			t.Errorf("expected tokenID %s, got %s", tokenID, token.ID)
		}
		return nil
	}

	user := userdomain.User{
		ID:       userdomain.ID(userID),
		Username: "testuser",
	}

	token, err := rotator.IssueRefreshToken(context.Background(), user)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token.RawToken == "" {
		t.Error("expected raw token to be set")
	}

	if token.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, token.UserID)
	}

	if token.ExpiresAt.Before(mockClock.Now()) {
		t.Error("expected expiration to be in the future")
	}
}

func TestRefreshTokenRotator_IssueRefreshToken_RotateWhenMaxReached(t *testing.T) {
	rotator, mockRefreshTokenRepo, mockIDGenerator, _ := setupRefreshTokenRotator(t)

	userID := "user-123"
	tokenID := "token-id-123"

	mockIDGenerator.newIDFunc = func() (string, error) {
		return tokenID, nil
	}

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		if uid != userID {
			t.Errorf("expected userID %s, got %s", userID, uid)
		}
		if maxTokens != constants.DefaultMaxRefreshTokensPerUser {
			t.Errorf("expected maxTokens %d, got %d", constants.DefaultMaxRefreshTokensPerUser, maxTokens)
		}
		return nil
	}

	mockRefreshTokenRepo.createFunc = func(ctx context.Context, token authdomain.RefreshToken) error {
		return nil
	}

	user := userdomain.User{
		ID:       userdomain.ID(userID),
		Username: "testuser",
	}

	token, err := rotator.IssueRefreshToken(context.Background(), user)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token.RawToken == "" {
		t.Error("expected raw token to be set")
	}
}

func TestRefreshTokenRotator_IssueRefreshToken_IDGenerationError(t *testing.T) {
	rotator, mockRefreshTokenRepo, mockIDGenerator, _ := setupRefreshTokenRotator(t)

	mockIDGenerator.newIDFunc = func() (string, error) {
		return "", errors.New("id generation error")
	}

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		return nil
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

func TestRefreshTokenRotator_IssueRefreshToken_CircuitBreakerOpen(t *testing.T) {
	rotator, mockRefreshTokenRepo, mockIDGenerator, _ := setupRefreshTokenRotator(t)

	mockIDGenerator.newIDFunc = func() (string, error) {
		return "token-id", nil
	}

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		return nil
	}

	mockRefreshTokenRepo.createFunc = func(ctx context.Context, token authdomain.RefreshToken) error {
		return commonerrors.ErrCircuitOpen
	}

	user := userdomain.User{
		ID:       "user-123",
		Username: "testuser",
	}

	_, err := rotator.IssueRefreshToken(context.Background(), user)

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, commonerrors.ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestRefreshTokenRotator_RotateIfNeeded_NoRotation(t *testing.T) {
	rotator, mockRefreshTokenRepo, _, _ := setupRefreshTokenRotator(t)

	userID := "user-123"

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		if uid != userID {
			t.Errorf("expected userID %s, got %s", userID, uid)
		}
		return nil
	}

	err := rotator.RotateIfNeeded(context.Background(), userID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRefreshTokenRotator_RotateIfNeeded_RotationNeeded(t *testing.T) {
	rotator, mockRefreshTokenRepo, _, _ := setupRefreshTokenRotator(t)

	userID := "user-123"

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		if uid != userID {
			t.Errorf("expected userID %s, got %s", userID, uid)
		}
		if maxTokens != constants.DefaultMaxRefreshTokensPerUser {
			t.Errorf("expected maxTokens %d, got %d", constants.DefaultMaxRefreshTokensPerUser, maxTokens)
		}
		return nil
	}

	err := rotator.RotateIfNeeded(context.Background(), userID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRefreshTokenRotator_RotateIfNeeded_CircuitBreakerOpen(t *testing.T) {
	rotator, mockRefreshTokenRepo, _, _ := setupRefreshTokenRotator(t)

	userID := "user-123"

	mockRefreshTokenRepo.deleteExcessByUserIDFunc = func(ctx context.Context, uid string, maxTokens int) error {
		return commonerrors.ErrCircuitOpen
	}

	err := rotator.RotateIfNeeded(context.Background(), userID)

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, commonerrors.ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestGenerateRefreshToken_Success(t *testing.T) {
	token, err := service.GenerateRefreshToken()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token == "" {
		t.Error("expected token to be set")
	}

	if len(token) != constants.RefreshTokenHexLength {
		t.Errorf("expected token length %d, got %d", constants.RefreshTokenHexLength, len(token))
	}
}

func TestHashRefreshToken_Success(t *testing.T) {
	token := "test-refresh-token-123"
	hash1 := service.HashRefreshToken(token)
	hash2 := service.HashRefreshToken(token)

	if hash1 == "" {
		t.Error("expected hash to be set")
	}

	if hash1 != hash2 {
		t.Error("expected same hash for same token")
	}

	if len(hash1) != constants.RefreshTokenHexLength {
		t.Errorf("expected hash length %d, got %d", constants.RefreshTokenHexLength, len(hash1))
	}
}

func TestHashRefreshToken_DifferentTokens(t *testing.T) {
	token1 := "test-refresh-token-123"
	token2 := "test-refresh-token-456"

	hash1 := service.HashRefreshToken(token1)
	hash2 := service.HashRefreshToken(token2)

	if hash1 == hash2 {
		t.Error("expected different hashes for different tokens")
	}
}
