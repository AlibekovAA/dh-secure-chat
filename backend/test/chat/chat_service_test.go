package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/service"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

func setupChatService(t *testing.T) (*service.ChatService, *mockUserRepo, *mockIdentityService) {
	t.Helper()
	mockRepo := newMockUserRepo()
	mockIdentity := newMockIdentityService()
	log, _ := logger.New("", "test", "info")
	svc := service.NewChatService(service.ChatServiceDeps{
		Repo:            mockRepo,
		IdentityService: mockIdentity,
		Log:             log,
	})
	return svc, mockRepo, mockIdentity
}

func TestChatService_GetMe_Success(t *testing.T) {
	svc, mockRepo, _ := setupChatService(t)
	userID := "user-123"
	username := "testuser"
	createdAt := time.Now()

	mockRepo.findByIDFunc = func(ctx context.Context, id userdomain.ID) (userdomain.User, error) {
		if string(id) != userID {
			t.Errorf("expected id %s, got %s", userID, id)
		}
		return userdomain.User{
			ID:           userdomain.ID(userID),
			Username:     username,
			PasswordHash: "hash",
			CreatedAt:    createdAt,
		}, nil
	}

	user, err := svc.GetMe(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.ID != userID {
		t.Errorf("expected ID %s, got %s", userID, user.ID)
	}
	if user.Username != username {
		t.Errorf("expected Username %s, got %s", username, user.Username)
	}
}

func TestChatService_GetMe_UserNotFound(t *testing.T) {
	svc, mockRepo, _ := setupChatService(t)
	mockRepo.findByIDFunc = func(ctx context.Context, id userdomain.ID) (userdomain.User, error) {
		return userdomain.User{}, userrepo.ErrUserNotFound
	}

	_, err := svc.GetMe(context.Background(), "user-123")
	if err == nil {
		t.Fatal("expected error")
	}
	de, ok := commonerrors.AsDomainError(err)
	if !ok || de.Code() != "USER_NOT_FOUND" {
		t.Errorf("expected USER_NOT_FOUND, got %v", err)
	}
}

func TestChatService_GetMe_RepoError(t *testing.T) {
	svc, mockRepo, _ := setupChatService(t)
	repoErr := errors.New("db connection failed")
	mockRepo.findByIDFunc = func(ctx context.Context, id userdomain.ID) (userdomain.User, error) {
		return userdomain.User{}, repoErr
	}

	_, err := svc.GetMe(context.Background(), "user-123")
	if err == nil {
		t.Fatal("expected error")
	}
	de, ok := commonerrors.AsDomainError(err)
	if !ok || de.Code() != "USER_GET_FAILED" {
		t.Errorf("expected USER_GET_FAILED, got %v", err)
	}
}

func TestChatService_SearchUsers_EmptyQuery(t *testing.T) {
	svc, _, _ := setupChatService(t)

	_, err := svc.SearchUsers(context.Background(), "  ", 10)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, commonerrors.ErrEmptyQuery) {
		t.Errorf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestChatService_SearchUsers_QueryTooLong(t *testing.T) {
	svc, _, _ := setupChatService(t)
	longQuery := string(make([]byte, constants.MaxSearchQueryLength+1))

	_, err := svc.SearchUsers(context.Background(), longQuery, 10)
	if err == nil {
		t.Fatal("expected error")
	}
	de, ok := commonerrors.AsDomainError(err)
	if !ok || de.Code() != "QUERY_TOO_LONG" {
		t.Errorf("expected QUERY_TOO_LONG, got %v", err)
	}
}

func TestChatService_SearchUsers_Success(t *testing.T) {
	svc, mockRepo, _ := setupChatService(t)
	createdAt := time.Now()
	mockRepo.searchByUsernameFunc = func(ctx context.Context, query string, limit int) ([]userdomain.Summary, error) {
		return []userdomain.Summary{
			{ID: "u1", Username: "user1", CreatedAt: createdAt},
			{ID: "u2", Username: "user2", CreatedAt: createdAt},
		}, nil
	}

	users, err := svc.SearchUsers(context.Background(), "user", 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
	if users[0].Username != "user1" || users[1].Username != "user2" {
		t.Errorf("unexpected usernames: %v", users)
	}
}

func TestChatService_SearchUsers_RepoError(t *testing.T) {
	svc, mockRepo, _ := setupChatService(t)
	repoErr := errors.New("search failed")
	mockRepo.searchByUsernameFunc = func(ctx context.Context, query string, limit int) ([]userdomain.Summary, error) {
		return nil, repoErr
	}

	_, err := svc.SearchUsers(context.Background(), "user", 10)
	if err == nil {
		t.Fatal("expected error")
	}
	de, ok := commonerrors.AsDomainError(err)
	if !ok || de.Code() != "USER_SEARCH_FAILED" {
		t.Errorf("expected USER_SEARCH_FAILED, got %v", err)
	}
}

func TestChatService_SearchUsers_ZeroLimitUsesDefault(t *testing.T) {
	svc, mockRepo, _ := setupChatService(t)
	var capturedLimit int
	mockRepo.searchByUsernameFunc = func(ctx context.Context, query string, limit int) ([]userdomain.Summary, error) {
		capturedLimit = limit
		return nil, nil
	}

	_, err := svc.SearchUsers(context.Background(), "user", 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedLimit != constants.DefaultSearchLimit {
		t.Errorf("expected limit %d, got %d", constants.DefaultSearchLimit, capturedLimit)
	}
}

func TestChatService_SearchUsers_LimitExceedsMaxClamped(t *testing.T) {
	svc, mockRepo, _ := setupChatService(t)
	var capturedLimit int
	mockRepo.searchByUsernameFunc = func(ctx context.Context, query string, limit int) ([]userdomain.Summary, error) {
		capturedLimit = limit
		return nil, nil
	}

	_, err := svc.SearchUsers(context.Background(), "user", constants.MaxSearchResultsLimit+100)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedLimit != constants.MaxSearchResultsLimit {
		t.Errorf("expected limit clamped to %d, got %d", constants.MaxSearchResultsLimit, capturedLimit)
	}
}

func TestChatService_GetIdentityKey_Success(t *testing.T) {
	svc, _, mockIdentity := setupChatService(t)
	pubKey := []byte("public-key-bytes")
	mockIdentity.getPublicKeyFunc = func(ctx context.Context, userID string) ([]byte, error) {
		return pubKey, nil
	}

	key, err := svc.GetIdentityKey(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(key) != string(pubKey) {
		t.Errorf("expected key %q, got %q", pubKey, key)
	}
}

func TestChatService_GetIdentityKey_NotFound(t *testing.T) {
	svc, _, mockIdentity := setupChatService(t)
	mockIdentity.getPublicKeyFunc = func(ctx context.Context, userID string) ([]byte, error) {
		return nil, commonerrors.ErrIdentityKeyNotFound
	}

	_, err := svc.GetIdentityKey(context.Background(), "user-123")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, commonerrors.ErrIdentityKeyNotFound) {
		t.Errorf("expected ErrIdentityKeyNotFound, got %v", err)
	}
}

func TestChatService_GetIdentityKey_IdentityError(t *testing.T) {
	svc, _, mockIdentity := setupChatService(t)
	identityErr := errors.New("identity service down")
	mockIdentity.getPublicKeyFunc = func(ctx context.Context, userID string) ([]byte, error) {
		return nil, identityErr
	}

	_, err := svc.GetIdentityKey(context.Background(), "user-123")
	if err == nil {
		t.Fatal("expected error")
	}
	de, ok := commonerrors.AsDomainError(err)
	if !ok || de.Code() != "IDENTITY_KEY_GET_FAILED" {
		t.Errorf("expected IDENTITY_KEY_GET_FAILED, got %v", err)
	}
}
