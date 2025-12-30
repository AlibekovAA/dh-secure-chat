package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

func TestTokenIssuer_IssueAccessToken_Success(t *testing.T) {
	mockIDGenerator := &mockIDGenerator{}
	mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))

	jti := "jti-123"
	mockIDGenerator.newIDFunc = func() (string, error) {
		return jti, nil
	}

	issuer := service.NewTokenIssuer(
		"test-secret-key-must-be-at-least-32-bytes-long",
		mockIDGenerator,
		15*time.Minute,
		mockClock,
	)

	user := userdomain.User{
		ID:       "user-123",
		Username: "testuser",
	}

	token, tokenJTI, err := issuer.IssueAccessToken(user)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token == "" {
		t.Error("expected token to be set")
	}

	if tokenJTI != jti {
		t.Errorf("expected jti %s, got %s", jti, tokenJTI)
	}
}

func TestTokenIssuer_IssueAccessToken_IDGenerationError(t *testing.T) {
	mockIDGenerator := &mockIDGenerator{}
	mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))

	mockIDGenerator.newIDFunc = func() (string, error) {
		return "", errors.New("id generation failed")
	}

	issuer := service.NewTokenIssuer(
		"test-secret-key-must-be-at-least-32-bytes-long",
		mockIDGenerator,
		15*time.Minute,
		mockClock,
	)

	user := userdomain.User{
		ID:       "user-123",
		Username: "testuser",
	}

	_, _, err := issuer.IssueAccessToken(user)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTokenIssuer_ParseToken_InvalidToken(t *testing.T) {
	mockIDGenerator := &mockIDGenerator{}
	mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))

	issuer := service.NewTokenIssuer(
		"test-secret-key-must-be-at-least-32-bytes-long",
		mockIDGenerator,
		15*time.Minute,
		mockClock,
	)

	_, err := issuer.ParseToken("invalid-token")

	if err == nil {
		t.Fatal("expected error")
	}
}
func TestTokenIssuer_ParseToken_WrongSecret(t *testing.T) {
	mockIDGenerator := &mockIDGenerator{}
	mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))

	jti := "jti-123"
	mockIDGenerator.newIDFunc = func() (string, error) {
		return jti, nil
	}

	issuer1 := service.NewTokenIssuer(
		"test-secret-key-must-be-at-least-32-bytes-long",
		mockIDGenerator,
		15*time.Minute,
		mockClock,
	)

	issuer2 := service.NewTokenIssuer(
		"different-secret-key-must-be-at-least-32-bytes",
		mockIDGenerator,
		15*time.Minute,
		mockClock,
	)

	user := userdomain.User{
		ID:       "user-123",
		Username: "testuser",
	}

	token, _, err := issuer1.IssueAccessToken(user)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	_, err = issuer2.ParseToken(token)

	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}
