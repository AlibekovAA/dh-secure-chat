package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	authcleanup "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/cleanup"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func TestStartCleanup_RefreshToken(t *testing.T) {
	mockRefreshTokenRepo := &mockRefreshTokenRepo{}
	deletedCount := constants.TestCleanupDeletedCount1

	mockRefreshTokenRepo.deleteExpiredFunc = func(ctx context.Context) (int64, error) {
		return deletedCount, nil
	}

	log, _ := logger.New("", "test", "info")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go authcleanup.StartRefreshTokenCleanup(ctx, mockRefreshTokenRepo, log)

	time.Sleep(constants.TestCleanupInitialDelay)
	cancel()
	time.Sleep(constants.TestCleanupWaitDelay)
}

func TestStartCleanup_RevokedToken(t *testing.T) {
	mockRevokedTokenRepo := &mockRevokedTokenRepo{}
	deletedCount := constants.TestCleanupDeletedCount2

	mockRevokedTokenRepo.deleteExpiredFunc = func(ctx context.Context) (int64, error) {
		return deletedCount, nil
	}

	log, _ := logger.New("", "test", "info")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go authcleanup.StartRevokedTokenCleanup(ctx, mockRevokedTokenRepo, log)

	time.Sleep(constants.TestCleanupInitialDelay)
	cancel()
	time.Sleep(constants.TestCleanupWaitDelay)
}

func TestStartCleanup_ErrorHandling(t *testing.T) {
	mockRefreshTokenRepo := &mockRefreshTokenRepo{}

	mockRefreshTokenRepo.deleteExpiredFunc = func(ctx context.Context) (int64, error) {
		return 0, errors.New("cleanup error")
	}

	log, _ := logger.New("", "test", "info")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go authcleanup.StartRefreshTokenCleanup(ctx, mockRefreshTokenRepo, log)

	time.Sleep(constants.TestCleanupInitialDelay)
	cancel()
	time.Sleep(constants.TestCleanupWaitDelay)
}

func TestStartCleanup_NoExpiredTokens(t *testing.T) {
	mockRefreshTokenRepo := &mockRefreshTokenRepo{}

	mockRefreshTokenRepo.deleteExpiredFunc = func(ctx context.Context) (int64, error) {
		return 0, nil
	}

	log, _ := logger.New("", "test", "info")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go authcleanup.StartRefreshTokenCleanup(ctx, mockRefreshTokenRepo, log)

	time.Sleep(constants.TestCleanupInitialDelay)
	cancel()
	time.Sleep(constants.TestCleanupWaitDelay)
}
