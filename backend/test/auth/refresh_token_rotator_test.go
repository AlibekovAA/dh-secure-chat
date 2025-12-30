package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

func setupRefreshTokenRotator(t *testing.T) (*service.RefreshTokenRotator, *mockRefreshTokenRepo, *mockIDGenerator, *clock.MockClock) {
	_ = t
	mockRefreshTokenRepo := &mockRefreshTokenRepo{}
	mockIDGenerator := &mockIDGenerator{}
	mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	log, _ := logger.New("", "test", "info")

	dbCB := db.NewDBCircuitBreaker(5, 5*time.Second, 30*time.Second, log)

	rotator := service.NewRefreshTokenRotator(
		mockRefreshTokenRepo,
		dbCB,
		mockIDGenerator,
		7*24*time.Hour,
		5,
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

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, nil
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

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 5, nil
	}

	mockRefreshTokenRepo.deleteOldestByUserIDFunc = func(ctx context.Context, uid string) error {
		if uid != userID {
			t.Errorf("expected userID %s, got %s", userID, uid)
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

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, nil
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

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, nil
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

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		if uid != userID {
			t.Errorf("expected userID %s, got %s", userID, uid)
		}
		return 3, nil
	}

	err := rotator.RotateIfNeeded(context.Background(), userID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRefreshTokenRotator_RotateIfNeeded_RotationNeeded(t *testing.T) {
	rotator, mockRefreshTokenRepo, _, _ := setupRefreshTokenRotator(t)

	userID := "user-123"

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 5, nil
	}

	mockRefreshTokenRepo.deleteOldestByUserIDFunc = func(ctx context.Context, uid string) error {
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

func TestRefreshTokenRotator_RotateIfNeeded_CircuitBreakerOpen(t *testing.T) {
	rotator, mockRefreshTokenRepo, _, _ := setupRefreshTokenRotator(t)

	userID := "user-123"

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, commonerrors.ErrCircuitOpen
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

	if len(token) != 64 {
		t.Errorf("expected token length 64, got %d", len(token))
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

	if len(hash1) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash1))
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
