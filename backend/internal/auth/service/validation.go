package service

import (
	"errors"
	"regexp"
	"unicode"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
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

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

func validateCredentials(username, password string) error {
	if len(username) < constants.UsernameMinLength || len(username) > constants.UsernameMaxLength {
		return ValidationError{reason: "username must be between 3 and 32 characters"}
	}

	if len(password) < constants.PasswordMinLength || len(password) > constants.PasswordMaxLength {
		return ValidationError{reason: "password must be between 8 and 72 characters"}
	}

	if !isValidUsername(username) {
		return ValidationError{reason: "username may contain only letters, digits, underscore and dash"}
	}

	if !isValidPassword(password) {
		return ValidationError{reason: "password must contain at least one letter and one digit"}
	}

	return nil
}

func isValidUsername(value string) bool {
	if !usernameRegex.MatchString(value) {
		return false
	}

	if !unicode.IsLetter(rune(value[0])) && !unicode.IsDigit(rune(value[0])) {
		return false
	}

	if !unicode.IsLetter(rune(value[len(value)-1])) && !unicode.IsDigit(rune(value[len(value)-1])) {
		return false
	}

	return true
}

func isValidPassword(value string) bool {
	hasLetter := false
	hasDigit := false

	for _, r := range value {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return true
		}
	}

	return hasLetter && hasDigit
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
