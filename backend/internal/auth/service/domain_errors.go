package service

import (
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

var (
	ErrInvalidCredentials = commonerrors.NewDomainError(
		"INVALID_CREDENTIALS",
		commonerrors.CategoryUnauthorized,
		401,
		"invalid username or password",
	)

	ErrUsernameTaken = commonerrors.NewDomainError(
		"USERNAME_TAKEN",
		commonerrors.CategoryConflict,
		409,
		"username already exists",
	)

	ErrValidation = commonerrors.NewDomainError(
		"VALIDATION_FAILED",
		commonerrors.CategoryValidation,
		400,
		"validation failed",
	)

	ErrInvalidRefreshToken = commonerrors.NewDomainError(
		"INVALID_REFRESH_TOKEN",
		commonerrors.CategoryUnauthorized,
		401,
		"invalid refresh token",
	)

	ErrRefreshTokenExpired = commonerrors.NewDomainError(
		"REFRESH_TOKEN_EXPIRED",
		commonerrors.CategoryUnauthorized,
		401,
		"refresh token expired",
	)

	ErrServiceUnavailable = commonerrors.NewDomainError(
		"SERVICE_UNAVAILABLE",
		commonerrors.CategoryExternal,
		503,
		"service temporarily unavailable",
	)
)
