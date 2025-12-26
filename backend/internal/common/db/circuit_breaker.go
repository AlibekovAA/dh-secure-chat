package db

import (
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/resilience"
)

type DBCircuitBreaker struct {
	*resilience.CircuitBreaker
}

func NewDBCircuitBreaker(threshold int32, timeout, resetAfter time.Duration, log *logger.Logger) *DBCircuitBreaker {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Threshold:  threshold,
		Timeout:    timeout,
		ResetAfter: resetAfter,
		Name:       "database",
		Logger:     log,
	})
	return &DBCircuitBreaker{CircuitBreaker: cb}
}
