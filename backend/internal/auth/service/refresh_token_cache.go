package service

import (
	"context"
	"sync"
	"time"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type refreshTokenCacheEntry struct {
	token     authdomain.RefreshToken
	userID    string
	expiresAt time.Time
}

type RefreshTokenCache struct {
	cache  sync.Map
	clock  clock.Clock
	log    *logger.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

func NewRefreshTokenCache(ctx context.Context, clock clock.Clock, log *logger.Logger) *RefreshTokenCache {
	cacheCtx, cancel := context.WithCancel(ctx)
	cache := &RefreshTokenCache{
		clock:  clock,
		log:    log,
		ctx:    cacheCtx,
		cancel: cancel,
	}

	go cache.cleanup()

	return cache
}

func (c *RefreshTokenCache) Get(hash string) (authdomain.RefreshToken, string, bool) {
	if entry, ok := c.cache.Load(hash); ok {
		e := entry.(*refreshTokenCacheEntry)
		if c.clock.Now().Before(e.expiresAt) {
			return e.token, e.userID, true
		}
		c.cache.Delete(hash)
	}
	return authdomain.RefreshToken{}, "", false
}

func (c *RefreshTokenCache) Set(hash string, token authdomain.RefreshToken, userID string) {
	entry := &refreshTokenCacheEntry{
		token:     token,
		userID:    userID,
		expiresAt: c.clock.Now().Add(constants.RefreshTokenCacheTTL),
	}
	c.cache.Store(hash, entry)
}

func (c *RefreshTokenCache) Invalidate(hash string) {
	c.cache.Delete(hash)
}

func (c *RefreshTokenCache) InvalidateByUserID(userID string) {
	c.cache.Range(func(key, value interface{}) bool {
		entry := value.(*refreshTokenCacheEntry)
		if entry.userID == userID {
			c.cache.Delete(key)
		}
		return true
	})
}

func (c *RefreshTokenCache) cleanup() {
	ticker := time.NewTicker(constants.RefreshTokenCacheCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			now := c.clock.Now()
			removed := 0
			c.cache.Range(func(key, value interface{}) bool {
				entry := value.(*refreshTokenCacheEntry)
				if now.After(entry.expiresAt) {
					c.cache.Delete(key)
					removed++
				}
				return true
			})
			if removed > 0 {
				c.log.Debugf("refresh token cache cleaned up %d expired entries", removed)
			}
		}
	}
}

func (c *RefreshTokenCache) Close() {
	c.cancel()
}
