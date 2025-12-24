package db

import (
	"context"
	"sync/atomic"
	"time"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	prommetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/prometheus"
)

type DBCircuitBreaker struct {
	failures    atomic.Int32
	lastFailure atomic.Value
	threshold   int32
	timeout     time.Duration
	resetAfter  time.Duration
	log         *logger.Logger
}

func NewDBCircuitBreaker(threshold int32, timeout, resetAfter time.Duration, log *logger.Logger) *DBCircuitBreaker {
	cb := &DBCircuitBreaker{
		threshold:  threshold,
		timeout:    timeout,
		resetAfter: resetAfter,
		log:        log,
	}
	cb.lastFailure.Store(time.Time{})
	return cb
}

func (cb *DBCircuitBreaker) isOpen() bool {
	if cb.failures.Load() < cb.threshold {
		prommetrics.CircuitBreakerState.WithLabelValues("database").Set(0)
		return false
	}

	lastFailure := cb.lastFailure.Load().(time.Time)
	if lastFailure.IsZero() {
		prommetrics.CircuitBreakerState.WithLabelValues("database").Set(0)
		return false
	}

	if time.Since(lastFailure) > cb.resetAfter {
		cb.reset()
		prommetrics.CircuitBreakerState.WithLabelValues("database").Set(0)
		return false
	}

	prommetrics.CircuitBreakerState.WithLabelValues("database").Set(1)
	return true
}

func (cb *DBCircuitBreaker) recordFailure() {
	cb.failures.Add(1)
	cb.lastFailure.Store(time.Now())
	prommetrics.CircuitBreakerFailures.WithLabelValues("database").Inc()
	cb.log.Warn("database circuit breaker: failure recorded")
}

func (cb *DBCircuitBreaker) reset() {
	cb.failures.Store(0)
	cb.lastFailure.Store(time.Time{})
}

func (cb *DBCircuitBreaker) Call(ctx context.Context, fn func(context.Context) error) error {
	if cb.isOpen() {
		cb.log.Warn("database circuit breaker: circuit is open, rejecting request")
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
