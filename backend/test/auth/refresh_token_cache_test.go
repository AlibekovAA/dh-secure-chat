package auth

import (
	"context"
	"testing"
	"time"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func setupRefreshTokenCache(t *testing.T) (*service.RefreshTokenCache, *clock.MockClock) {
	_ = t
	mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	log, _ := logger.New("", "test", "info")

	ctx := context.Background()
	cache := service.NewRefreshTokenCache(ctx, mockClock, log)

	return cache, mockClock
}

func TestRefreshTokenCache_Get_Miss(t *testing.T) {
	cache, _ := setupRefreshTokenCache(t)

	token, userID, found := cache.Get("nonexistent-hash")

	if found {
		t.Error("expected cache miss")
	}

	if token.ID != "" {
		t.Error("expected empty token")
	}

	if userID != "" {
		t.Error("expected empty userID")
	}
}

func TestRefreshTokenCache_Get_Hit(t *testing.T) {
	cache, mockClock := setupRefreshTokenCache(t)

	hash := "test-hash-123"
	userID := "user-123"
	token := authdomain.RefreshToken{
		ID:        "token-id",
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: mockClock.Now().Add(constants.TestTokenExpiryOffset),
		CreatedAt: mockClock.Now(),
	}

	cache.Set(hash, token, userID)

	retrievedToken, retrievedUserID, found := cache.Get(hash)

	if !found {
		t.Error("expected cache hit")
	}

	if retrievedToken.ID != token.ID {
		t.Errorf("expected token ID %s, got %s", token.ID, retrievedToken.ID)
	}

	if retrievedUserID != userID {
		t.Errorf("expected userID %s, got %s", userID, retrievedUserID)
	}
}

func TestRefreshTokenCache_Get_Expired(t *testing.T) {
	cache, mockClock := setupRefreshTokenCache(t)

	hash := "test-hash-123"
	userID := "user-123"
	token := authdomain.RefreshToken{
		ID:        "token-id",
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: mockClock.Now().Add(constants.TestTokenExpiryOffset),
		CreatedAt: mockClock.Now(),
	}

	cache.Set(hash, token, userID)

	mockClock.SetTime(mockClock.Now().Add(constants.RefreshTokenCacheTTL + time.Second))

	retrievedToken, retrievedUserID, found := cache.Get(hash)

	if found {
		t.Error("expected cache miss for expired token")
	}

	if retrievedToken.ID != "" {
		t.Error("expected empty token")
	}

	if retrievedUserID != "" {
		t.Error("expected empty userID")
	}
}

func TestRefreshTokenCache_Invalidate(t *testing.T) {
	cache, mockClock := setupRefreshTokenCache(t)

	hash := "test-hash-123"
	userID := "user-123"
	token := authdomain.RefreshToken{
		ID:        "token-id",
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: mockClock.Now().Add(constants.TestTokenExpiryOffset),
		CreatedAt: mockClock.Now(),
	}

	cache.Set(hash, token, userID)
	cache.Invalidate(hash)

	retrievedToken, retrievedUserID, found := cache.Get(hash)

	if found {
		t.Error("expected cache miss after invalidation")
	}

	if retrievedToken.ID != "" {
		t.Error("expected empty token")
	}

	if retrievedUserID != "" {
		t.Error("expected empty userID")
	}
}

func TestRefreshTokenCache_InvalidateByUserID(t *testing.T) {
	cache, mockClock := setupRefreshTokenCache(t)

	hash1 := "test-hash-1"
	hash2 := "test-hash-2"
	userID1 := "user-123"
	userID2 := "user-456"

	token1 := authdomain.RefreshToken{
		ID:        "token-id-1",
		TokenHash: hash1,
		UserID:    userID1,
		ExpiresAt: mockClock.Now().Add(constants.TestTokenExpiryOffset),
		CreatedAt: mockClock.Now(),
	}

	token2 := authdomain.RefreshToken{
		ID:        "token-id-2",
		TokenHash: hash2,
		UserID:    userID2,
		ExpiresAt: mockClock.Now().Add(constants.TestTokenExpiryOffset),
		CreatedAt: mockClock.Now(),
	}

	cache.Set(hash1, token1, userID1)
	cache.Set(hash2, token2, userID2)

	cache.InvalidateByUserID(userID1)

	_, _, found1 := cache.Get(hash1)
	if found1 {
		t.Error("expected cache miss for userID1 after invalidation")
	}

	_, _, found2 := cache.Get(hash2)
	if !found2 {
		t.Error("expected cache hit for userID2")
	}
}

func TestRefreshTokenCache_Close(t *testing.T) {
	cache, _ := setupRefreshTokenCache(t)

	cache.Close()

	cache.Close()
}
