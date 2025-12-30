package auth

import (
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

	if validationErr, ok := service.AsValidationError(err); !ok {
		t.Error("expected ValidationError")
	} else {
		if validationErr.Error() != "username must be between 3 and 32 characters" {
			t.Errorf("expected specific error message, got %s", validationErr.Error())
		}
	}
}

func TestCredentialValidator_Validate_UsernameTooLong(t *testing.T) {
	validator := service.NewCredentialValidator()

	longUsername := "a" + string(make([]byte, 32))
	err := validator.Validate(longUsername, "password123")

	if err == nil {
		t.Fatal("expected validation error")
	}

	if validationErr, ok := service.AsValidationError(err); !ok {
		t.Error("expected ValidationError")
	} else {
		if validationErr.Error() != "username must be between 3 and 32 characters" {
			t.Errorf("expected specific error message, got %s", validationErr.Error())
		}
	}
}

func TestCredentialValidator_Validate_PasswordTooShort(t *testing.T) {
	validator := service.NewCredentialValidator()

	err := validator.Validate("testuser", "pass123")

	if err == nil {
		t.Fatal("expected validation error")
	}

	if validationErr, ok := service.AsValidationError(err); !ok {
		t.Error("expected ValidationError")
	} else {
		if validationErr.Error() != "password must be between 8 and 72 characters" {
			t.Errorf("expected specific error message, got %s", validationErr.Error())
		}
	}
}

func TestCredentialValidator_Validate_PasswordTooLong(t *testing.T) {
	validator := service.NewCredentialValidator()

	longPassword := string(make([]byte, 73))
	err := validator.Validate("testuser", longPassword)

	if err == nil {
		t.Fatal("expected validation error")
	}

	if validationErr, ok := service.AsValidationError(err); !ok {
		t.Error("expected ValidationError")
	} else {
		if validationErr.Error() != "password must be between 8 and 72 characters" {
			t.Errorf("expected specific error message, got %s", validationErr.Error())
		}
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
			if validationErr, ok := service.AsValidationError(err); !ok {
				t.Error("expected ValidationError")
			} else {
				if validationErr.Error() != "username may contain only letters, digits, underscore and dash" {
					t.Errorf("expected specific error message, got %s", validationErr.Error())
				}
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
			if validationErr, ok := service.AsValidationError(err); !ok {
				t.Error("expected ValidationError")
			} else {
				if validationErr.Error() != "username may contain only letters, digits, underscore and dash" {
					t.Errorf("expected specific error message, got %s", validationErr.Error())
				}
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

	if validationErr, ok := service.AsValidationError(err); !ok {
		t.Error("expected ValidationError")
	} else {
		if validationErr.Error() != "password must contain at least one letter and one digit" {
			t.Errorf("expected specific error message, got %s", validationErr.Error())
		}
	}
}

func TestCredentialValidator_Validate_PasswordWithoutDigit(t *testing.T) {
	validator := service.NewCredentialValidator()

	err := validator.Validate("testuser", "abcdefgh")

	if err == nil {
		t.Fatal("expected validation error")
	}

	if validationErr, ok := service.AsValidationError(err); !ok {
		t.Error("expected ValidationError")
	} else {
		if validationErr.Error() != "password must contain at least one letter and one digit" {
			t.Errorf("expected specific error message, got %s", validationErr.Error())
		}
	}
}

func TestAsValidationError_WithValidationError(t *testing.T) {
	validator := service.NewCredentialValidator()
	err := validator.Validate("ab", "password123")

	validationErr, ok := service.AsValidationError(err)
	if !ok {
		t.Fatal("expected ValidationError")
	}

	if validationErr.Error() == "" {
		t.Error("expected error message")
	}
}

func TestAsValidationError_WithDomainError(t *testing.T) {
	domainErr := commonerrors.NewDomainError(
		"VALIDATION_FAILED",
		commonerrors.CategoryValidation,
		400,
		"test validation error",
	)

	validationErr, ok := service.AsValidationError(domainErr)
	if !ok {
		t.Fatal("expected ValidationError")
	}

	if validationErr.Error() != "test validation error" {
		t.Errorf("expected error message 'test validation error', got %s", validationErr.Error())
	}
}

func TestAsValidationError_WithOtherError(t *testing.T) {
	err := commonerrors.NewDomainError(
		"OTHER_ERROR",
		commonerrors.CategoryInternal,
		500,
		"other error",
	)

	_, ok := service.AsValidationError(err)
	if ok {
		t.Error("expected false for non-validation error")
	}
}
