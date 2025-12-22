package websocket

import (
	"context"
	"sync/atomic"
	"time"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	prommetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/prometheus"
)

type CircuitBreaker struct {
	failures    atomic.Int32
	lastFailure atomic.Value
	threshold   int32
	timeout     time.Duration
	resetAfter  time.Duration
	name        string
}

func NewCircuitBreaker(threshold int32, timeout, resetAfter time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		threshold:  threshold,
		timeout:    timeout,
		resetAfter: resetAfter,
		name:       "last_seen_update",
	}
	cb.lastFailure.Store(time.Time{})
	return cb
}

func (cb *CircuitBreaker) isOpen() bool {
	if cb.failures.Load() < cb.threshold {
		prommetrics.CircuitBreakerState.WithLabelValues(cb.name).Set(0)
		return false
	}

	lastFailure := cb.lastFailure.Load().(time.Time)
	if lastFailure.IsZero() {
		prommetrics.CircuitBreakerState.WithLabelValues(cb.name).Set(0)
		return false
	}

	if time.Since(lastFailure) > cb.resetAfter {
		cb.reset()
		prommetrics.CircuitBreakerState.WithLabelValues(cb.name).Set(0)
		return false
	}

	prommetrics.CircuitBreakerState.WithLabelValues(cb.name).Set(1)
	return true
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failures.Add(1)
	cb.lastFailure.Store(time.Now())
	prommetrics.CircuitBreakerFailures.WithLabelValues(cb.name).Inc()
}

func (cb *CircuitBreaker) reset() {
	cb.failures.Store(0)
	cb.lastFailure.Store(time.Time{})
}

func (cb *CircuitBreaker) Call(ctx context.Context, fn func(context.Context) error) error {
	if cb.isOpen() {
		return commonerrors.ErrCircuitOpen
	}

	callCtx, cancel := context.WithTimeout(ctx, cb.timeout)
	defer cancel()

	err := fn(callCtx)
	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.reset()
	return nil
}
