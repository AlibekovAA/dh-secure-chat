package service

import (
	"errors"
	"net/http"

	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

func handleCircuitBreakerError(err error) error {
	if errors.Is(err, commonerrors.ErrCircuitOpen) {
		return ErrServiceUnavailable.WithCause(err)
	}
	return err
}

func handleRefreshTokenError(err error) error {
	if errors.Is(err, ErrRefreshTokenExpired) {
		return ErrRefreshTokenExpired
	}
	if errors.Is(err, authrepo.ErrRefreshTokenNotFound) {
		return ErrInvalidRefreshToken
	}
	return err
}

func newInternalError(code, message string, cause error) commonerrors.DomainError {
	err := commonerrors.NewDomainError(
		code,
		commonerrors.CategoryInternal,
		http.StatusInternalServerError,
		message,
	)
	if cause != nil {
		err = err.WithCause(cause)
	}
	return err
}
