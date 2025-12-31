package resilience

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	pgx "github.com/jackc/pgx/v4"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

type CircuitBreaker struct {
	failures    atomic.Int32
	lastFailure atomic.Value
	threshold   int32
	timeout     time.Duration
	resetAfter  time.Duration
	name        string
	log         *logger.Logger
}

type CircuitBreakerConfig struct {
	Threshold  int32
	Timeout    time.Duration
	ResetAfter time.Duration
	Name       string
	Logger     *logger.Logger
}

func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		threshold:  config.Threshold,
		timeout:    config.Timeout,
		resetAfter: config.ResetAfter,
		name:       config.Name,
		log:        config.Logger,
	}
	cb.lastFailure.Store(time.Time{})
	return cb
}

func (cb *CircuitBreaker) IsOpen() bool {
	if cb.failures.Load() < cb.threshold {
		cb.setState(0)
		return false
	}

	lastFailure := cb.lastFailure.Load().(time.Time)
	if lastFailure.IsZero() {
		cb.setState(0)
		return false
	}

	if time.Since(lastFailure) > cb.resetAfter {
		cb.reset()
		cb.setState(0)
		return false
	}

	cb.setState(1)
	return true
}

func (cb *CircuitBreaker) setState(state float64) {
	if cb.name != "" {
		metrics.CircuitBreakerState.WithLabelValues(cb.name).Set(state)
	}
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failures.Add(1)
	cb.lastFailure.Store(time.Now())
	if cb.name != "" {
		metrics.CircuitBreakerFailures.WithLabelValues(cb.name).Inc()
	}
	if cb.log != nil {
		cb.log.Warnf("circuit breaker [%s]: failure recorded", cb.name)
	}
}

func (cb *CircuitBreaker) reset() {
	cb.failures.Store(0)
	cb.lastFailure.Store(time.Time{})
}

func (cb *CircuitBreaker) Call(ctx context.Context, fn func(context.Context) error) error {
	return cb.CallWithFallback(ctx, fn, nil)
}

func (cb *CircuitBreaker) CallWithFallback(ctx context.Context, fn func(context.Context) error, fallback func() error) error {
	if cb.IsOpen() {
		if cb.log != nil {
			if fallback != nil {
				cb.log.Warnf("circuit breaker [%s]: circuit is open, using fallback", cb.name)
			} else {
				cb.log.Warnf("circuit breaker [%s]: circuit is open, rejecting request", cb.name)
			}
		}
		if fallback != nil {
			return fallback()
		}
		return commonerrors.ErrCircuitOpen
	}

	callCtx, cancel := context.WithTimeout(ctx, cb.timeout)
	defer cancel()

	err := fn(callCtx)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			cb.recordFailure()
		}
		if fallback != nil {
			if cb.log != nil {
				cb.log.Infof("circuit breaker [%s]: operation failed, using fallback", cb.name)
			}
			return fallback()
		}
		return err
	}

	cb.reset()
	return nil
}
