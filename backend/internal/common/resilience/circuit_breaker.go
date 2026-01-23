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

type CircuitBreakerInterface interface {
	Call(ctx context.Context, fn func(context.Context) error) error
}

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
	breaker := &CircuitBreaker{
		threshold:  config.Threshold,
		timeout:    config.Timeout,
		resetAfter: config.ResetAfter,
		name:       config.Name,
		log:        config.Logger,
	}
	breaker.lastFailure.Store(time.Time{})
	return breaker
}

func (breaker *CircuitBreaker) IsOpen() bool {
	if breaker.failures.Load() < breaker.threshold {
		breaker.setState(0)
		return false
	}

	lastFailure := breaker.lastFailure.Load().(time.Time)
	if lastFailure.IsZero() {
		breaker.setState(0)
		return false
	}

	if time.Since(lastFailure) > breaker.resetAfter {
		breaker.reset()
		breaker.setState(0)
		return false
	}

	breaker.setState(1)
	return true
}

func (breaker *CircuitBreaker) setState(state float64) {
	if breaker.name != "" {
		metrics.CircuitBreakerState.WithLabelValues(breaker.name).Set(state)
	}
}

func (breaker *CircuitBreaker) recordFailure() {
	breaker.failures.Add(1)
	breaker.lastFailure.Store(time.Now())
	if breaker.log != nil {
		breaker.log.Warnf("circuit breaker [%s]: failure recorded", breaker.name)
	}
}

func (breaker *CircuitBreaker) reset() {
	breaker.failures.Store(0)
	breaker.lastFailure.Store(time.Time{})
}

func (breaker *CircuitBreaker) Call(ctx context.Context, fn func(context.Context) error) error {
	return breaker.CallWithFallback(ctx, fn, nil)
}

func (breaker *CircuitBreaker) CallWithFallback(ctx context.Context, fn func(context.Context) error, fallback func() error) error {
	if breaker.IsOpen() {
		if breaker.log != nil {
			if fallback != nil {
				breaker.log.Warnf("circuit breaker [%s]: circuit is open, using fallback", breaker.name)
			} else {
				breaker.log.Warnf("circuit breaker [%s]: circuit is open, rejecting request", breaker.name)
			}
		}
		if fallback != nil {
			return fallback()
		}
		return commonerrors.ErrCircuitOpen
	}

	callCtx, cancel := context.WithTimeout(ctx, breaker.timeout)
	defer cancel()

	err := fn(callCtx)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			breaker.recordFailure()
		}
		if fallback != nil {
			if breaker.log != nil {
				breaker.log.Infof("circuit breaker [%s]: operation failed, using fallback", breaker.name)
			}
			return fallback()
		}
		return err
	}

	breaker.reset()
	return nil
}
