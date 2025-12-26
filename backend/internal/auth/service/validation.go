package service

import (
	"errors"
	"unicode"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

type ValidationError struct {
	reason string
}

func (e ValidationError) Error() string {
	return e.reason
}

func (e ValidationError) Unwrap() error {
	return ErrValidation
}

type CredentialValidator struct{}

func NewCredentialValidator() *CredentialValidator {
	return &CredentialValidator{}
}

func (cv *CredentialValidator) Validate(username, password string) error {
	return validateCredentials(username, password)
}

func validateCredentials(username, password string) error {
	if len(username) < 3 || len(username) > 32 {
		return ValidationError{reason: "username must be between 3 and 32 characters"}
	}

	if len(password) < 8 || len(password) > 72 {
		return ValidationError{reason: "password must be between 8 and 72 characters"}
	}

	if !isSafeUsername(username) {
		return ValidationError{reason: "username may contain only letters, digits, underscore and dash"}
	}

	return nil
}

func isSafeUsername(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		if r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func AsValidationError(err error) (ValidationError, bool) {
	var v ValidationError
	if errors.As(err, &v) {
		return v, true
	}
	if domainErr, ok := commonerrors.AsDomainError(err); ok && domainErr.Code() == "VALIDATION_FAILED" {
		return ValidationError{reason: domainErr.Message()}, true
	}
	return ValidationError{}, false
}
