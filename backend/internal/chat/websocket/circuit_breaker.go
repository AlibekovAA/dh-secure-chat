package websocket

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type CircuitBreaker struct {
	failures    atomic.Int32
	lastFailure atomic.Value
	threshold   int32
	timeout     time.Duration
	resetAfter  time.Duration
}

func NewCircuitBreaker(threshold int32, timeout, resetAfter time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		threshold:  threshold,
		timeout:    timeout,
		resetAfter: resetAfter,
	}
	cb.lastFailure.Store(time.Time{})
	return cb
}

func (cb *CircuitBreaker) isOpen() bool {
	if cb.failures.Load() < cb.threshold {
		return false
	}

	lastFailure := cb.lastFailure.Load().(time.Time)
	if lastFailure.IsZero() {
		return false
	}

	if time.Since(lastFailure) > cb.resetAfter {
		cb.reset()
		return false
	}

	return true
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failures.Add(1)
	cb.lastFailure.Store(time.Now())
}

func (cb *CircuitBreaker) reset() {
	cb.failures.Store(0)
	cb.lastFailure.Store(time.Time{})
}

func (cb *CircuitBreaker) Call(ctx context.Context, fn func(context.Context) error) error {
	if cb.isOpen() {
		return ErrCircuitOpen
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
