package auth

import (
	"context"
	"errors"
	"testing"

	authcleanup "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/cleanup"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func TestRunCleanupOnce_RefreshToken(t *testing.T) {
	mockRefreshTokenRepo := &mockRefreshTokenRepo{}
	deletedCount := constants.TestCleanupDeletedCount1

	mockRefreshTokenRepo.deleteExpiredFunc = func(ctx context.Context) (int64, error) {
		return deletedCount, nil
	}

	log, err := logger.New(t.TempDir(), "test", "info")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	if got := authcleanup.RunCleanupOnce(context.Background(), mockRefreshTokenRepo, log, "refresh token"); got != deletedCount {
		t.Fatalf("expected %d deleted, got %d", deletedCount, got)
	}
}

func TestStartCleanup_RevokedToken(t *testing.T) {
	mockRevokedTokenRepo := &mockRevokedTokenRepo{}
	deletedCount := constants.TestCleanupDeletedCount2

	mockRevokedTokenRepo.deleteExpiredFunc = func(ctx context.Context) (int64, error) {
		return deletedCount, nil
	}

	log, err := logger.New(t.TempDir(), "test", "info")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	if got := authcleanup.RunCleanupOnce(context.Background(), mockRevokedTokenRepo, log, "revoked token"); got != deletedCount {
		t.Fatalf("expected %d deleted, got %d", deletedCount, got)
	}
}

func TestStartCleanup_ErrorHandling(t *testing.T) {
	mockRefreshTokenRepo := &mockRefreshTokenRepo{}

	mockRefreshTokenRepo.deleteExpiredFunc = func(ctx context.Context) (int64, error) {
		return 0, errors.New("cleanup error")
	}

	log, err := logger.New(t.TempDir(), "test", "info")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	if got := authcleanup.RunCleanupOnce(context.Background(), mockRefreshTokenRepo, log, "refresh token"); got != 0 {
		t.Fatalf("expected zero after error, got %d", got)
	}
}

func TestStartCleanup_NoExpiredTokens(t *testing.T) {
	mockRefreshTokenRepo := &mockRefreshTokenRepo{}

	mockRefreshTokenRepo.deleteExpiredFunc = func(ctx context.Context) (int64, error) {
		return 0, nil
	}

	log, err := logger.New(t.TempDir(), "test", "info")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	if got := authcleanup.RunCleanupOnce(context.Background(), mockRefreshTokenRepo, log, "refresh token"); got != 0 {
		t.Fatalf("expected zero, got %d", got)
	}
}
