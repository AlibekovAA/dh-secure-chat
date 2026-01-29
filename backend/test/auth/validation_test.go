package auth

import (
	"net/http"
	"testing"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

func TestCredentialValidator_Validate_Success(t *testing.T) {
	validator := service.NewCredentialValidator()

	testCases := []struct {
		name     string
		username string
		password string
	}{
		{"valid credentials 1", "testuser", "password123"},
		{"valid credentials 2", "user123", "pass1234"},
		{"valid credentials 3", "test_user", "password1"},
		{"valid credentials 4", "test-user", "pass1234"},
		{"valid credentials 5", "a1b2c3", "password123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Validate(tc.username, tc.password)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestCredentialValidator_Validate_UsernameTooShort(t *testing.T) {
	validator := service.NewCredentialValidator()

	err := validator.Validate("ab", "password123")

	if err == nil {
		t.Fatal("expected validation error")
	}

	de, ok := commonerrors.AsDomainError(err)
	if !ok || de.Code() != "VALIDATION_USERNAME_LENGTH" {
		t.Errorf("expected VALIDATION_USERNAME_LENGTH, got %v", err)
	}
}

func TestCredentialValidator_Validate_UsernameTooLong(t *testing.T) {
	validator := service.NewCredentialValidator()

	longUsername := "a" + string(make([]byte, 32))
	err := validator.Validate(longUsername, "password123")

	if err == nil {
		t.Fatal("expected validation error")
	}

	de, ok := commonerrors.AsDomainError(err)
	if !ok || de.Code() != "VALIDATION_USERNAME_LENGTH" {
		t.Errorf("expected VALIDATION_USERNAME_LENGTH, got %v", err)
	}
}

func TestCredentialValidator_Validate_PasswordTooShort(t *testing.T) {
	validator := service.NewCredentialValidator()

	err := validator.Validate("testuser", "pass123")

	if err == nil {
		t.Fatal("expected validation error")
	}

	de, ok := commonerrors.AsDomainError(err)
	if !ok || de.Code() != "VALIDATION_PASSWORD_LENGTH" {
		t.Errorf("expected VALIDATION_PASSWORD_LENGTH, got %v", err)
	}
}

func TestCredentialValidator_Validate_PasswordTooLong(t *testing.T) {
	validator := service.NewCredentialValidator()

	longPassword := string(make([]byte, 73))
	err := validator.Validate("testuser", longPassword)

	if err == nil {
		t.Fatal("expected validation error")
	}

	de, ok := commonerrors.AsDomainError(err)
	if !ok || de.Code() != "VALIDATION_PASSWORD_LENGTH" {
		t.Errorf("expected VALIDATION_PASSWORD_LENGTH, got %v", err)
	}
}

func TestCredentialValidator_Validate_InvalidUsernameChars(t *testing.T) {
	validator := service.NewCredentialValidator()

	testCases := []struct {
		name     string
		username string
	}{
		{"special chars", "test@user"},
		{"spaces", "test user"},
		{"dots", "test.user"},
		{"unicode", "тестuser"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Validate(tc.username, "password123")
			if err == nil {
				t.Error("expected validation error")
			}
			de, ok := commonerrors.AsDomainError(err)
			if !ok || de.Code() != "VALIDATION_USERNAME_CHARS" {
				t.Errorf("expected VALIDATION_USERNAME_CHARS, got %v", err)
			}
		})
	}
}

func TestCredentialValidator_Validate_UsernameStartsOrEndsWithInvalidChar(t *testing.T) {
	validator := service.NewCredentialValidator()

	testCases := []struct {
		name     string
		username string
	}{
		{"starts with dash", "-testuser"},
		{"starts with underscore", "_testuser"},
		{"ends with dash", "testuser-"},
		{"ends with underscore", "testuser_"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Validate(tc.username, "password123")
			if err == nil {
				t.Error("expected validation error")
			}
			de, ok := commonerrors.AsDomainError(err)
			if !ok || de.Code() != "VALIDATION_USERNAME_CHARS" {
				t.Errorf("expected VALIDATION_USERNAME_CHARS, got %v", err)
			}
		})
	}
}

func TestCredentialValidator_Validate_PasswordWithoutLetter(t *testing.T) {
	validator := service.NewCredentialValidator()

	err := validator.Validate("testuser", "12345678")

	if err == nil {
		t.Fatal("expected validation error")
	}

	de, ok := commonerrors.AsDomainError(err)
	if !ok || de.Code() != "VALIDATION_PASSWORD_LATIN_DIGIT" {
		t.Errorf("expected VALIDATION_PASSWORD_LATIN_DIGIT, got %v", err)
	}
}

func TestCredentialValidator_Validate_PasswordWithoutDigit(t *testing.T) {
	validator := service.NewCredentialValidator()

	err := validator.Validate("testuser", "abcdefgh")

	if err == nil {
		t.Fatal("expected validation error")
	}

	de, ok := commonerrors.AsDomainError(err)
	if !ok || de.Code() != "VALIDATION_PASSWORD_LATIN_DIGIT" {
		t.Errorf("expected VALIDATION_PASSWORD_LATIN_DIGIT, got %v", err)
	}
}

func TestCredentialValidator_Validate_ReturnsDomainError(t *testing.T) {
	validator := service.NewCredentialValidator()
	err := validator.Validate("ab", "password123")

	de, ok := commonerrors.AsDomainError(err)
	if !ok {
		t.Fatal("expected domain error")
	}
	if de.Code() == "" {
		t.Error("expected non-empty code")
	}
	if de.HTTPStatus() != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", de.HTTPStatus())
	}
}
