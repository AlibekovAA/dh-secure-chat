package websocket

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	observabilitymetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type presenceCacheEntry struct {
	exists    bool
	expiresAt time.Time
}

type PresenceService struct {
	sender         MessageSender
	userRepo       userrepo.Repository
	existenceCache sync.Map
	log            *logger.Logger
	clock          clock.Clock
	ctx            context.Context
}

type PresenceServiceDeps struct {
	Sender   MessageSender
	UserRepo userrepo.Repository
	Log      *logger.Logger
	Clock    clock.Clock
}

func NewPresenceService(ctx context.Context, deps PresenceServiceDeps) *PresenceService {
	return &PresenceService{
		sender:   deps.Sender,
		userRepo: deps.UserRepo,
		log:      deps.Log,
		clock:    deps.Clock,
		ctx:      ctx,
	}
}

func (s *PresenceService) CheckUserExists(ctx context.Context, userID string) (bool, error) {
	if cached, ok := s.existenceCache.Load(userID); ok {
		entry := cached.(*presenceCacheEntry)
		if s.clock.Now().Before(entry.expiresAt) {
			observabilitymetrics.ChatWebSocketUserExistenceCacheHits.Inc()
			return entry.exists, nil
		}
		s.existenceCache.Delete(userID)
	}

	observabilitymetrics.ChatWebSocketUserExistenceCacheMisses.Inc()

	_, err := s.userRepo.FindByID(ctx, userdomain.ID(userID))
	exists := err == nil
	if err != nil && !errors.Is(err, userrepo.ErrUserNotFound) {
		return false, err
	}

	s.existenceCache.Store(userID, &presenceCacheEntry{
		exists:    exists,
		expiresAt: s.clock.Now().Add(constants.UserExistenceCacheTTL),
	})

	return exists, nil
}

func (s *PresenceService) SendPeerOffline(ctx context.Context, fromUserID, peerID string) error {
	msg, err := marshalMessage(TypePeerOffline, PeerOfflinePayload{PeerID: peerID})
	if err != nil {
		return commonerrors.ErrMarshalError.WithCause(err)
	}
	if err := s.sender.SendToUserWithContext(ctx, fromUserID, msg); err != nil {
		return err
	}
	return nil
}

func (s *PresenceService) StartCleanup() {
	ticker := time.NewTicker(constants.UserExistenceCacheCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			now := s.clock.Now()
			removed := 0
			total := 0
			s.existenceCache.Range(func(key, value interface{}) bool {
				total++
				entry := value.(*presenceCacheEntry)
				if now.After(entry.expiresAt) {
					s.existenceCache.Delete(key)
					removed++
				}
				return true
			})
			observabilitymetrics.ChatWebSocketUserExistenceCacheSize.Set(float64(total))
			if removed > 0 {
				s.log.Debugf("websocket cleaned up stale user existence cache entries count=%d", removed)
			}
		}
	}
}
