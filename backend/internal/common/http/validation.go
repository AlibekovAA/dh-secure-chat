package http

import (
	"strings"

	"github.com/google/uuid"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

func ValidateUUID(s string) error {
	if s == "" {
		return commonerrors.ErrEmptyUUID
	}
	_, err := uuid.Parse(s)
	return err
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

func ExtractAndValidateUserID(path string, suffix string) (string, error) {
	userID, ok := ExtractUserIDFromPath(path)
	if !ok || userID == "" {
		return "", commonerrors.ErrEmptyUUID
	}

	if suffix != "" {
		userID = strings.TrimSuffix(userID, suffix)
	}

	if err := ValidateUUID(userID); err != nil {
		return "", err
	}

	return userID, nil
}
