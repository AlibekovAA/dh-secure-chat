package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgconn"
)

func TestCircuitBreakerCountsOnlyTransientFailures(t *testing.T) {
	breaker := NewCircuitBreaker(CircuitBreakerConfig{Threshold: 1, Timeout: time.Second, ResetAfter: time.Minute})
	domainError := errors.New("username already exists")
	if err := breaker.Call(context.Background(), func(context.Context) error { return domainError }); !errors.Is(err, domainError) {
		t.Fatal(err)
	}
	if breaker.IsOpen() {
		t.Fatal("domain error opened circuit")
	}
	connectionError := &pgconn.PgError{Code: "08006", Message: "connection failure"}
	_ = breaker.Call(context.Background(), func(context.Context) error { return connectionError })
	if !breaker.IsOpen() {
		t.Fatal("transient connection failure did not open circuit")
	}

	timeoutBreaker := NewCircuitBreaker(CircuitBreakerConfig{Threshold: 1, Timeout: time.Second, ResetAfter: time.Minute})
	_ = timeoutBreaker.Call(context.Background(), func(context.Context) error { return context.DeadlineExceeded })
	if !timeoutBreaker.IsOpen() {
		t.Fatal("infrastructure timeout did not open circuit")
	}

	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()
	canceledBreaker := NewCircuitBreaker(CircuitBreakerConfig{Threshold: 1, Timeout: time.Second, ResetAfter: time.Minute})
	_ = canceledBreaker.Call(canceledContext, func(ctx context.Context) error { return ctx.Err() })
	if canceledBreaker.IsOpen() {
		t.Fatal("caller cancellation opened circuit")
	}
}
