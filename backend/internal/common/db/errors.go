package db

import (
	"errors"
	"fmt"
	"strings"
	"time"

	pgx "github.com/jackc/pgx/v4"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

func extractTableFromOperation(operation string) string {
	operation = strings.ToLower(operation)

	if strings.Contains(operation, "revoked") {
		return "revoked_tokens"
	}
	if strings.Contains(operation, "refresh") {
		return "refresh_tokens"
	}
	if strings.Contains(operation, "identity") || strings.Contains(operation, "key") {
		return "identity_keys"
	}
	if strings.Contains(operation, "token") {
		return "refresh_tokens"
	}
	if strings.Contains(operation, "user") {
		return "users"
	}
	return "unknown"
}

func handleError(err error, notFoundErr error, operation string, startTime time.Time) error {
	table := extractTableFromOperation(operation)
	duration := time.Since(startTime).Seconds()
	metrics.DBQueryDurationSeconds.WithLabelValues(operation, table).Observe(duration)

	if err == nil {
		return nil
	}
	if notFoundErr != nil && errors.Is(err, pgx.ErrNoRows) {
		return notFoundErr
	}
	errorType := strings.TrimPrefix(strings.TrimPrefix(fmt.Sprintf("%T", err), "*"), "pgx.")
	metrics.DBQueryErrors.WithLabelValues(operation, table, errorType).Inc()
	return commonerrors.ErrDatabaseError.WithCause(err)
}

func HandleQueryError(err error, notFoundErr error, operation string, startTime time.Time) error {
	return handleError(err, notFoundErr, operation, startTime)
}

func HandleExecError(err error, operation string, startTime time.Time) error {
	return handleError(err, nil, operation, startTime)
}

func MeasureQueryDuration(operation string, startTime time.Time) {
	table := extractTableFromOperation(operation)
	duration := time.Since(startTime).Seconds()
	metrics.DBQueryDurationSeconds.WithLabelValues(operation, table).Observe(duration)
}
