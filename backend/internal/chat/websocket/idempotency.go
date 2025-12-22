package websocket

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	prommetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/prometheus"
)

type idempotencyResult struct {
	result    interface{}
	err       error
	expiresAt time.Time
}

type IdempotencyTracker struct {
	operations sync.Map
	ttl        time.Duration
}

func NewIdempotencyTracker(ttl time.Duration) *IdempotencyTracker {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	tracker := &IdempotencyTracker{
		ttl: ttl,
	}

	go tracker.cleanup()

	return tracker
}

func (t *IdempotencyTracker) generateOperationID(userID string, msgType MessageType, payload []byte) string {
	hash := sha256.Sum256(append(append([]byte(userID), []byte(msgType)...), payload...))
	return hex.EncodeToString(hash[:])
}

func (t *IdempotencyTracker) Execute(operationID string, msgType MessageType, fn func() (interface{}, error)) (interface{}, error) {
	if result, ok := t.operations.Load(operationID); ok {
		res := result.(*idempotencyResult)
		if time.Now().Before(res.expiresAt) {
			prommetrics.ChatWebSocketIdempotencyDuplicates.WithLabelValues(string(msgType)).Inc()
			return res.result, res.err
		}
		t.operations.Delete(operationID)
	}

	result, err := fn()
	t.operations.Store(operationID, &idempotencyResult{
		result:    result,
		err:       err,
		expiresAt: time.Now().Add(t.ttl),
	})

	return result, err
}

func (t *IdempotencyTracker) cleanup() {
	ticker := time.NewTicker(t.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		t.operations.Range(func(key, value interface{}) bool {
			res := value.(*idempotencyResult)
			if now.After(res.expiresAt) {
				t.operations.Delete(key)
			}
			return true
		})
	}
}
