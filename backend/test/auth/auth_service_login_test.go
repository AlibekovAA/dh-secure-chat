package auth

import (
	"context"
	"errors"
	"testing"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

func TestAuthService_Login_Success(t *testing.T) {
	svc, mockUserRepo, _, mockRefreshTokenRepo, _, mockHasher, _, mockClock := setupAuthService(t)

	username := "testuser"
	password := "password123"
	hashedPassword := "hashed_password123"
	userID := "user-123"

	mockUserRepo.findByUsernameFunc = func(ctx context.Context, uname string) (userdomain.User, error) {
		if uname != username {
			t.Errorf("expected username %s, got %s", username, uname)
		}
		return userdomain.User{
			ID:           userdomain.ID(userID),
			Username:     username,
			PasswordHash: hashedPassword,
			CreatedAt:    mockClock.Now(),
		}, nil
	}

	mockHasher.compareFunc = func(hash string, pwd string) error {
		if hash != hashedPassword || pwd != password {
			return errors.New("password mismatch")
		}
		return nil
	}

	mockRefreshTokenRepo.createFunc = func(ctx context.Context, token authdomain.RefreshToken) error {
		return nil
	}

	mockRefreshTokenRepo.countByUserIDFunc = func(ctx context.Context, uid string) (int, error) {
		return 0, nil
	}

	result, err := svc.Login(context.Background(), service.LoginInput{
		Username: username,
		Password: password,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.AccessToken == "" {
		t.Error("expected access token to be set")
	}

	if result.RefreshToken == "" {
		t.Error("expected refresh token to be set")
	}
}

func TestAuthService_Login_ValidationError(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupAuthService(t)

	_, err := svc.Login(context.Background(), service.LoginInput{
		Username: "ab",
		Password: "pass123",
	})

	if err == nil {
		t.Fatal("expected validation error")
	}

	if domainErr, ok := commonerrors.AsDomainError(err); !ok || domainErr.Code() != "VALIDATION_FAILED" {
		t.Errorf("expected VALIDATION_FAILED error, got %v", err)
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	svc, mockUserRepo, _, _, _, _, _, _ := setupAuthService(t)

	mockUserRepo.findByUsernameFunc = func(ctx context.Context, username string) (userdomain.User, error) {
		return userdomain.User{}, userrepo.ErrUserNotFound
	}

	_, err := svc.Login(context.Background(), service.LoginInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	svc, mockUserRepo, _, _, _, mockHasher, _, mockClock := setupAuthService(t)

	mockUserRepo.findByUsernameFunc = func(ctx context.Context, username string) (userdomain.User, error) {
		return userdomain.User{
			ID:           "user-123",
			Username:     "testuser",
			PasswordHash: "hashed",
			CreatedAt:    mockClock.Now(),
		}, nil
	}

	mockHasher.compareFunc = func(hash string, password string) error {
		return errors.New("password mismatch")
	}

	_, err := svc.Login(context.Background(), service.LoginInput{
		Username: "testuser",
		Password: "wrongpass123",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_CircuitBreakerOpen(t *testing.T) {
	svc, mockUserRepo, _, _, _, _, _, _ := setupAuthService(t)

	mockUserRepo.findByUsernameFunc = func(ctx context.Context, username string) (userdomain.User, error) {
		return userdomain.User{}, commonerrors.ErrCircuitOpen
	}

	_, err := svc.Login(context.Background(), service.LoginInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if domainErr, ok := commonerrors.AsDomainError(err); !ok || domainErr.Code() != "SERVICE_UNAVAILABLE" {
		t.Errorf("expected SERVICE_UNAVAILABLE error, got %v", err)
	}
}

func TestAuthService_Login_DatabaseError(t *testing.T) {
	svc, mockUserRepo, _, _, _, _, _, _ := setupAuthService(t)

	mockUserRepo.findByUsernameFunc = func(ctx context.Context, username string) (userdomain.User, error) {
		return userdomain.User{}, errors.New("database connection error")
	}

	_, err := svc.Login(context.Background(), service.LoginInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if domainErr, ok := commonerrors.AsDomainError(err); !ok || domainErr.Code() != "DB_ERROR" {
		t.Errorf("expected DB_ERROR error, got %v", err)
	}
}
