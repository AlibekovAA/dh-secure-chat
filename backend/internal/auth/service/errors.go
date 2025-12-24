package service

import (
	"fmt"
)

type AuthError struct {
	Code    string
	Message string
	Err     error
}

func (e *AuthError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

var (
	ErrInvalidCredentials = &AuthError{
		Code:    "INVALID_CREDENTIALS",
		Message: "invalid username or password",
	}
	ErrUsernameTaken = &AuthError{
		Code:    "USERNAME_TAKEN",
		Message: "username already exists",
	}
	ErrValidation = &AuthError{
		Code:    "VALIDATION_FAILED",
		Message: "validation failed",
	}
	ErrInvalidRefreshToken = &AuthError{
		Code:    "INVALID_REFRESH_TOKEN",
		Message: "invalid refresh token",
	}
	ErrRefreshTokenExpired = &AuthError{
		Code:    "REFRESH_TOKEN_EXPIRED",
		Message: "refresh token expired",
	}
)
