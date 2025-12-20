package db

import (
	"errors"
	"fmt"

	pgx "github.com/jackc/pgx/v4"
)

func HandleQueryError(err error, notFoundErr error, operation string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFoundErr
	}
	return fmt.Errorf("failed to %s: %w", operation, err)
}

func HandleExecError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("failed to %s: %w", operation, err)
}
