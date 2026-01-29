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

	ErrValidationUsernameLength = commonerrors.NewDomainError(
		"VALIDATION_USERNAME_LENGTH",
		commonerrors.CategoryValidation,
		400,
		"username must be between 3 and 32 characters",
	)

	ErrValidationPasswordLength = commonerrors.NewDomainError(
		"VALIDATION_PASSWORD_LENGTH",
		commonerrors.CategoryValidation,
		400,
		"password must be between 8 and 72 characters",
	)

	ErrValidationUsernameChars = commonerrors.NewDomainError(
		"VALIDATION_USERNAME_CHARS",
		commonerrors.CategoryValidation,
		400,
		"username may contain only letters, digits, underscore and dash",
	)

	ErrValidationPasswordLatinDigit = commonerrors.NewDomainError(
		"VALIDATION_PASSWORD_LATIN_DIGIT",
		commonerrors.CategoryValidation,
		400,
		"password must contain at least one letter and one digit",
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
