package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/resilience"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type LastSeenUpdater struct {
	ctx            context.Context
	cancel         context.CancelFunc
	repo           userrepo.Repository
	log            *logger.Logger
	circuitBreaker *resilience.CircuitBreaker
	clock          clock.Clock
	updateInterval time.Duration
	queue          chan string
	lastSeenCache  map[string]time.Time
	mu             sync.Mutex
	wg             sync.WaitGroup
}

func NewLastSeenUpdater(ctx context.Context, repo userrepo.Repository, log *logger.Logger, updateInterval time.Duration, circuitBreaker *resilience.CircuitBreaker, clock clock.Clock) *LastSeenUpdater {
	updateCtx, cancel := context.WithCancel(ctx)
	updater := &LastSeenUpdater{
		ctx:            updateCtx,
		cancel:         cancel,
		repo:           repo,
		log:            log,
		circuitBreaker: circuitBreaker,
		clock:          clock,
		updateInterval: updateInterval,
		queue:          make(chan string, constants.LastSeenQueueSize),
		lastSeenCache:  make(map[string]time.Time),
	}

	updater.wg.Add(1)
	go updater.run()

	return updater
}

func (u *LastSeenUpdater) Enqueue(userID string) {
	now := u.clock.Now()

	u.mu.Lock()
	if last, ok := u.lastSeenCache[userID]; ok && now.Sub(last) < u.updateInterval {
		u.mu.Unlock()
		return
	}
	u.lastSeenCache[userID] = now
	u.mu.Unlock()

	select {
	case u.queue <- userID:
	default:
		u.log.WithFields(context.Background(), logger.Fields{
			"user_id": userID,
			"action":  "last_seen_enqueue_dropped",
		}).Warn("last seen queue is full, dropping update")
	}
}

func (u *LastSeenUpdater) Stop() {
	u.cancel()
	u.wg.Wait()
}

func (u *LastSeenUpdater) run() {
	defer u.wg.Done()

	ticker := time.NewTicker(constants.LastSeenFlushEvery)
	defer ticker.Stop()

	pending := make(map[string]struct{})

	for {
		select {
		case <-u.ctx.Done():
			u.flush(pending)
			return
		case userID := <-u.queue:
			pending[userID] = struct{}{}
			if len(pending) >= constants.LastSeenBatchSize {
				u.flush(pending)
			}
		case <-ticker.C:
			u.flush(pending)
		}
	}
}

func (u *LastSeenUpdater) flush(pending map[string]struct{}) {
	if len(pending) == 0 {
		return
	}

	ids := make([]userdomain.ID, 0, len(pending))
	for id := range pending {
		ids = append(ids, userdomain.ID(id))
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.LastSeenUpdateTimeout)
	defer cancel()

	var err error
	if u.circuitBreaker != nil {
		err = u.circuitBreaker.CallWithFallback(ctx, func(callCtx context.Context) error {
			return u.repo.UpdateLastSeenBatch(callCtx, ids)
		}, func() error {
			u.log.WithFields(ctx, logger.Fields{
				"count":  len(ids),
				"action": "last_seen_batch_skipped",
			}).Debug("last_seen update skipped: circuit breaker is open")
			return nil
		})
	} else {
		err = u.repo.UpdateLastSeenBatch(ctx, ids)
	}

	if err != nil {
		u.log.WithFields(ctx, logger.Fields{
			"count":  len(ids),
			"action": "last_seen_batch_failed",
		}).Warnf("websocket failed to batch update last_seen: %v", err)
	}

	for id := range pending {
		delete(pending, id)
	}
}
