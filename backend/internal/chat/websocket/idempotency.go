package websocket

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

type idempotencyResult struct {
	result    interface{}
	err       error
	expiresAt time.Time
}

type IdempotencyTracker struct {
	operations sync.Map
	ttl        time.Duration
	clock      clock.Clock
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewIdempotencyTracker(ctx context.Context, ttl time.Duration, clock clock.Clock) *IdempotencyTracker {
	if ttl <= 0 {
		ttl = constants.IdempotencyTTL
	}

	trackerCtx, cancel := context.WithCancel(ctx)
	tracker := &IdempotencyTracker{
		ttl:    ttl,
		clock:  clock,
		ctx:    trackerCtx,
		cancel: cancel,
	}

	go tracker.cleanup()

	return tracker
}

func (t *IdempotencyTracker) GenerateOperationID(userID string, msgType MessageType, payload []byte) string {
	hash := sha256.Sum256(append(append([]byte(userID), []byte(msgType)...), payload...))
	return hex.EncodeToString(hash[:])
}

func (t *IdempotencyTracker) Execute(operationID string, msgType MessageType, fn func() (interface{}, error)) (interface{}, error) {
	if result, ok := t.operations.Load(operationID); ok {
		res := result.(*idempotencyResult)
		if t.clock.Now().Before(res.expiresAt) {
			metrics.ChatWebSocketIdempotencyDuplicates.WithLabelValues(string(msgType)).Inc()
			return res.result, res.err
		}
		t.operations.Delete(operationID)
	}

	result, err := fn()
	t.operations.Store(operationID, &idempotencyResult{
		result:    result,
		err:       err,
		expiresAt: t.clock.Now().Add(t.ttl),
	})

	return result, err
}

func (t *IdempotencyTracker) cleanup() {
	ticker := time.NewTicker(t.ttl / 2)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			now := t.clock.Now()
			removed := 0
			t.operations.Range(func(key, value interface{}) bool {
				res := value.(*idempotencyResult)
				if now.After(res.expiresAt) {
					t.operations.Delete(key)
					removed++
				}
				return true
			})
		}
	}
}

func (t *IdempotencyTracker) Shutdown() {
	t.cancel()
}

type IdempotencyAdapter struct {
	Tracker *IdempotencyTracker
}

func (a *IdempotencyAdapter) GenerateOperationID(userID string, msgType string, payload []byte) string {
	return a.Tracker.GenerateOperationID(userID, MessageType(msgType), payload)
}

func (a *IdempotencyAdapter) Execute(operationID string, msgType string, fn func() (interface{}, error)) (interface{}, error) {
	return a.Tracker.Execute(operationID, MessageType(msgType), fn)
}
