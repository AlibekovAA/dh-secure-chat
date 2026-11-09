package websocket

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	done      chan struct{}
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
	for {
		candidate := &idempotencyResult{done: make(chan struct{})}
		loadedValue, loaded := t.operations.LoadOrStore(operationID, candidate)
		if !loaded {
			candidate.result, candidate.err = fn()
			candidate.expiresAt = t.clock.Now().Add(t.ttl)
			if errors.Is(candidate.err, ErrProcessorQueueFull) || errors.Is(candidate.err, ErrProcessorClosed) {
				t.operations.Delete(operationID)
			}
			close(candidate.done)
			return candidate.result, candidate.err
		}

		res := loadedValue.(*idempotencyResult)
		<-res.done
		if t.clock.Now().Before(res.expiresAt) {
			metrics.ChatWebSocketIdempotencyDuplicates.WithLabelValues(string(msgType)).Inc()
			return res.result, res.err
		}
		t.operations.CompareAndDelete(operationID, res)
	}
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
			t.operations.Range(func(key, value interface{}) bool {
				res := value.(*idempotencyResult)
				select {
				case <-res.done:
				default:
					return true
				}
				if now.After(res.expiresAt) {
					t.operations.CompareAndDelete(key, res)
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
