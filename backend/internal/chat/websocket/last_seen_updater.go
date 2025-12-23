package websocket

import (
	"context"
	"errors"
	"sync"
	"time"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

const (
	lastSeenQueueSize     = 100
	lastSeenBatchSize     = 100
	lastSeenFlushEvery    = 500 * time.Millisecond
	lastSeenUpdateTimeout = 3 * time.Second
)

type LastSeenUpdater struct {
	ctx            context.Context
	cancel         context.CancelFunc
	repo           userrepo.Repository
	log            *logger.Logger
	circuitBreaker *CircuitBreaker
	updateInterval time.Duration
	queue          chan string
	lastSeenCache  map[string]time.Time
	mu             sync.Mutex
	wg             sync.WaitGroup
}

func NewLastSeenUpdater(ctx context.Context, repo userrepo.Repository, log *logger.Logger, updateInterval time.Duration, circuitBreaker *CircuitBreaker) *LastSeenUpdater {
	updateCtx, cancel := context.WithCancel(ctx)
	updater := &LastSeenUpdater{
		ctx:            updateCtx,
		cancel:         cancel,
		repo:           repo,
		log:            log,
		circuitBreaker: circuitBreaker,
		updateInterval: updateInterval,
		queue:          make(chan string, lastSeenQueueSize),
		lastSeenCache:  make(map[string]time.Time),
	}

	updater.wg.Add(1)
	go updater.run()

	return updater
}

func (u *LastSeenUpdater) Enqueue(userID string) {
	now := time.Now()

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

	ticker := time.NewTicker(lastSeenFlushEvery)
	defer ticker.Stop()

	pending := make(map[string]struct{})

	for {
		select {
		case <-u.ctx.Done():
			u.flush(pending)
			return
		case userID := <-u.queue:
			pending[userID] = struct{}{}
			if len(pending) >= lastSeenBatchSize {
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

	ctx, cancel := context.WithTimeout(context.Background(), lastSeenUpdateTimeout)
	defer cancel()

	var err error
	if u.circuitBreaker != nil {
		err = u.circuitBreaker.Call(ctx, func(callCtx context.Context) error {
			return u.repo.UpdateLastSeenBatch(callCtx, ids)
		})
	} else {
		err = u.repo.UpdateLastSeenBatch(ctx, ids)
	}

	if err != nil && !errors.Is(err, commonerrors.ErrCircuitOpen) {
		u.log.WithFields(ctx, logger.Fields{
			"count":  len(ids),
			"action": "last_seen_batch_failed",
		}).Warnf("websocket failed to batch update last_seen: %v", err)
	}

	for id := range pending {
		delete(pending, id)
	}
}
