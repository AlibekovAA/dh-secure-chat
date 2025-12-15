package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type AuthService struct {
	repo      repository.UserRepository
	jwtSecret []byte
	now       func() time.Time
}

func NewAuthService(repo repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		now:       time.Now,
	}
}

type RegisterInput struct {
	Username string
	Password string
}

type LoginInput struct {
	Username string
	Password string
}

type AuthResult struct {
	Token string
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	log := logger.GetInstance()

	log.Infof("register attempt for username=%s", input.Username)

	if err := validateCredentials(input.Username, input.Password); err != nil {
		if vErr, ok := AsValidationError(err); ok {
			log.Warnf("register validation failed for username=%s: %s", input.Username, vErr.Error())
		} else {
			log.Warnf("register validation failed for username=%s: %v", input.Username, err)
		}
		return AuthResult{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Errorf("failed to hash password for username=%s: %v", input.Username, err)
		return AuthResult{}, err
	}

	user := domain.User{
		ID:           domain.UserID(generateUUID()),
		Username:     input.Username,
		PasswordHash: string(hash),
		CreatedAt:    s.now(),
	}

	err = s.repo.Create(ctx, user)
	if err != nil {
		if errors.Is(err, repository.ErrUsernameAlreadyExists) {
			log.Infof("register failed for username=%s: username already exists", input.Username)
			return AuthResult{}, ErrUsernameTaken
		}
		log.Errorf("register failed for username=%s: %v", input.Username, err)
		return AuthResult{}, err
	}

	token, err := s.issueToken(user)
	if err != nil {
		log.Errorf("failed to issue token for username=%s: %v", input.Username, err)
		return AuthResult{}, err
	}

	log.Infof("register successful for username=%s user_id=%s", user.Username, user.ID)

	return AuthResult{Token: token}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	log := logger.GetInstance()

	log.Infof("login attempt for username=%s", input.Username)

	if err := validateCredentials(input.Username, input.Password); err != nil {
		if vErr, ok := AsValidationError(err); ok {
			log.Warnf("login validation failed for username=%s: %s", input.Username, vErr.Error())
		} else {
			log.Warnf("login validation failed for username=%s: %v", input.Username, err)
		}
		return AuthResult{}, err
	}

	user, err := s.repo.FindByUsername(ctx, input.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			log.Infof("login failed for username=%s: user not found", input.Username)
			return AuthResult{}, ErrInvalidCredentials
		}
		log.Errorf("login failed for username=%s: %v", input.Username, err)
		return AuthResult{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		log.Infof("login failed for username=%s: invalid password", input.Username)
		return AuthResult{}, ErrInvalidCredentials
	}

	token, err := s.issueToken(user)
	if err != nil {
		log.Errorf("failed to issue token for username=%s: %v", input.Username, err)
		return AuthResult{}, err
	}

	log.Infof("login successful for username=%s user_id=%s", user.Username, user.ID)

	return AuthResult{Token: token}, nil
}

func (s *AuthService) issueToken(user domain.User) (string, error) {
	claims := jwt.MapClaims{
		"sub": string(user.ID),
		"usr": user.Username,
		"exp": s.now().Add(24 * time.Hour).Unix(),
		"iat": s.now().Unix(),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}
