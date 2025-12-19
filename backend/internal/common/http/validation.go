package http

import (
	"errors"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput = errors.New("invalid input")
)

func ValidateUUID(s string) error {
	if s == "" {
		return errors.New("uuid cannot be empty")
	}
	_, err := uuid.Parse(s)
	return err
}

func ValidateString(s string, minLen, maxLen int, name string) error {
	if len(s) < minLen {
		return errors.New(name + " is too short")
	}
	if len(s) > maxLen {
		return errors.New(name + " is too long")
	}
	return nil
}

func ValidateSafeString(s string, minLen, maxLen int, name string) error {
	if err := ValidateString(s, minLen, maxLen, name); err != nil {
		return err
	}

	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '.' {
			return errors.New(name + " contains invalid characters")
		}
	}

	return nil
}

func ValidateFilename(filename string) error {
	if filename == "" {
		return errors.New("filename cannot be empty")
	}

	if len(filename) > 255 {
		return errors.New("filename is too long")
	}

	if strings.Contains(filename, "..") {
		return errors.New("filename contains invalid path")
	}

	for _, char := range []rune{'/', '\\', '<', '>', ':', '"', '|', '?', '*', '\x00'} {
		if strings.ContainsRune(filename, char) {
			return errors.New("filename contains invalid characters")
		}
	}

	return nil
}

func ValidateMimeType(mimeType string) error {
	if mimeType == "" {
		return errors.New("mime type cannot be empty")
	}

	parts := strings.Split(mimeType, "/")
	if len(parts) != 2 {
		return errors.New("invalid mime type format")
	}

	for _, part := range parts {
		if len(part) == 0 || len(part) > 127 {
			return errors.New("invalid mime type format")
		}
		for _, r := range part {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' {
				return errors.New("invalid mime type format")
			}
		}
	}

	return nil
}

func ValidateFileSize(size int64, maxSize int64) error {
	if size < 0 {
		return errors.New("file size cannot be negative")
	}
	if size > maxSize {
		return errors.New("file size exceeds maximum allowed size")
	}
	return nil
}

func ExtractUserIDFromPath(path string) (string, bool) {
	var prefix, userID string

	if strings.HasPrefix(path, "/api/identity/users/") {
		prefix = "/api/identity/users/"
	} else if strings.HasPrefix(path, "/api/chat/users/") {
		prefix = "/api/chat/users/"
	} else {
		return "", false
	}

	remaining := strings.TrimPrefix(path, prefix)
	parts := strings.Split(remaining, "/")
	if len(parts) > 0 && parts[0] != "" {
		userID = parts[0]
		return userID, true
	}

	return "", false
}
