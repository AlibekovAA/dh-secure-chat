package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	pgx "github.com/jackc/pgx/v4"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

var DefaultRetryConfig = RetryConfig{
	MaxAttempts:  3,
	InitialDelay: 100 * time.Millisecond,
	MaxDelay:     2 * time.Second,
	Multiplier:   2.0,
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "08000", "08003", "08006", "08001", "08004", "08007", "08P01":
			return true
		case "40001", "40P01":
			return true
		case "55P03":
			return true
		}
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}

	return false
}

func RetryWithBackoff(ctx context.Context, log *logger.Logger, config RetryConfig, operation func() error) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := operation()
		if err == nil {
			if attempt > 1 {
				log.Infof("database operation succeeded after %d attempts", attempt)
			}
			return nil
		}

		lastErr = err

		if !isRetryableError(err) {
			return err
		}

		if attempt == config.MaxAttempts {
			break
		}

		log.Warnf("database operation failed (attempt %d/%d): %v, retrying in %v", attempt, config.MaxAttempts, err, delay)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		case <-time.After(delay):
		}

		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return fmt.Errorf("database operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}
