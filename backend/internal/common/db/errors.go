package db

import (
	"errors"
	"fmt"
	"strings"
	"time"

	pgx "github.com/jackc/pgx/v4"

	prommetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/prometheus"
)

func extractTableFromOperation(operation string) string {
	operation = strings.ToLower(operation)
	if strings.Contains(operation, "user") {
		return "users"
	}
	if strings.Contains(operation, "identity") || strings.Contains(operation, "key") {
		return "identity_keys"
	}
	if strings.Contains(operation, "refresh") || strings.Contains(operation, "token") {
		return "refresh_tokens"
	}
	if strings.Contains(operation, "revoked") {
		return "revoked_tokens"
	}
	return "unknown"
}

func HandleQueryError(err error, notFoundErr error, operation string, startTime time.Time) error {
	table := extractTableFromOperation(operation)
	duration := time.Since(startTime).Seconds()
	prommetrics.DBQueryDurationSeconds.WithLabelValues(operation, table).Observe(duration)

	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFoundErr
	}
	errorType := fmt.Sprintf("%T", err)
	prommetrics.DBQueryErrors.WithLabelValues(operation, table, errorType).Inc()
	return fmt.Errorf("failed to %s: %w", operation, err)
}

func HandleExecError(err error, operation string, startTime time.Time) error {
	table := extractTableFromOperation(operation)
	duration := time.Since(startTime).Seconds()
	prommetrics.DBQueryDurationSeconds.WithLabelValues(operation, table).Observe(duration)

	if err == nil {
		return nil
	}
	errorType := fmt.Sprintf("%T", err)
	prommetrics.DBQueryErrors.WithLabelValues(operation, table, errorType).Inc()
	return fmt.Errorf("failed to %s: %w", operation, err)
}

func MeasureQueryDuration(operation string, startTime time.Time) {
	table := extractTableFromOperation(operation)
	duration := time.Since(startTime).Seconds()
	prommetrics.DBQueryDurationSeconds.WithLabelValues(operation, table).Observe(duration)
}
