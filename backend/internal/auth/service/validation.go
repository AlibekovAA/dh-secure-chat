package service

import (
	"regexp"
	"unicode"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
)

type CredentialValidator struct{}

func NewCredentialValidator() CredentialValidator {
	return CredentialValidator{}
}

func (cv CredentialValidator) Validate(username, password string) error {
	return validateCredentials(username, password)
}

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

func validateCredentials(username, password string) error {
	if len(username) < constants.UsernameMinLength || len(username) > constants.UsernameMaxLength {
		return ErrValidationUsernameLength
	}

	if len(password) < constants.PasswordMinLength || len(password) > constants.PasswordMaxLength {
		return ErrValidationPasswordLength
	}

	if !isValidUsername(username) {
		return ErrValidationUsernameChars
	}

	if !isValidPassword(password) {
		return ErrValidationPasswordLatinDigit
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

	return false
}
