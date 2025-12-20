package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	commoncrypto "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/crypto"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	identityservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type AuthService struct {
	repo             userrepo.Repository
	identityService  *identityservice.IdentityService
	refreshTokenRepo authrepo.RefreshTokenRepository
	revokedTokenRepo authrepo.RevokedTokenRepository
	hasher           commoncrypto.PasswordHasher
	idGenerator      commoncrypto.IDGenerator
	jwtSecret        []byte
	now              func() time.Time
	log              *logger.Logger
}

func NewAuthService(
	repo userrepo.Repository,
	identityService *identityservice.IdentityService,
	refreshTokenRepo authrepo.RefreshTokenRepository,
	revokedTokenRepo authrepo.RevokedTokenRepository,
	hasher commoncrypto.PasswordHasher,
	idGenerator commoncrypto.IDGenerator,
	jwtSecret string,
	log *logger.Logger,
) *AuthService {
	return &AuthService{
		repo:             repo,
		identityService:  identityService,
		refreshTokenRepo: refreshTokenRepo,
		revokedTokenRepo: revokedTokenRepo,
		hasher:           hasher,
		idGenerator:      idGenerator,
		jwtSecret:        []byte(jwtSecret),
		now:              time.Now,
		log:              log,
	}
}

type RegisterInput struct {
	Username       string
	Password       string
	IdentityPubKey []byte
}

type LoginInput struct {
	Username string
	Password string
}

type AuthResult struct {
	AccessToken      string
	RefreshToken     string
	RefreshExpiresAt time.Time
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	s.log.Infof("register attempt username=%s", input.Username)

	if err := validateCredentials(input.Username, input.Password); err != nil {
		s.log.Warnf("register validation failed username=%s: %v", input.Username, err)
		return AuthResult{}, err
	}

	hash, err := s.hasher.Hash(input.Password)
	if err != nil {
		s.log.Errorf("register failed username=%s: password hash error: %v", input.Username, err)
		return AuthResult{}, err
	}

	id, err := s.idGenerator.NewID()
	if err != nil {
		s.log.Errorf("register failed username=%s: id generation error: %v", input.Username, err)
		return AuthResult{}, err
	}

	user := userdomain.User{
		ID:           userdomain.ID(id),
		Username:     input.Username,
		PasswordHash: hash,
		CreatedAt:    s.now(),
	}

	err = s.repo.Create(ctx, user)
	if err != nil {
		if errors.Is(err, userrepo.ErrUsernameAlreadyExists) {
			s.log.Warnf("register failed username=%s: already exists", input.Username)
			return AuthResult{}, ErrUsernameTaken
		}
		s.log.Errorf("register failed username=%s: %v", input.Username, err)
		return AuthResult{}, &AuthError{
			Code:    "DB_ERROR",
			Message: "failed to create user",
			Err:     err,
		}
	}

	if len(input.IdentityPubKey) > 0 {
		if err := s.identityService.CreateIdentityKey(ctx, string(user.ID), input.IdentityPubKey); err != nil {
			if delErr := s.repo.Delete(ctx, user.ID); delErr != nil {
				s.log.Errorf("register failed username=%s: failed to delete user after identity key error: %v", input.Username, delErr)
			}
			if errors.Is(err, identityservice.ErrInvalidPublicKey) {
				return AuthResult{}, err
			}
			s.log.Errorf("register failed username=%s: failed to save identity key: %v", input.Username, err)
			return AuthResult{}, fmt.Errorf("failed to save identity key: %w", err)
		}
	}

	accessToken, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		s.log.Errorf("register failed username=%s: token issue error: %v", input.Username, err)
		return AuthResult{}, err
	}

	s.log.Infof("register success username=%s user_id=%s", user.Username, user.ID)

	return AuthResult{
		AccessToken:      accessToken,
		RefreshToken:     refresh.RawToken,
		RefreshExpiresAt: refresh.ExpiresAt,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	s.log.Infof("login attempt username=%s", input.Username)

	if err := validateCredentials(input.Username, input.Password); err != nil {
		s.log.Warnf("login validation failed username=%s: %v", input.Username, err)
		return AuthResult{}, err
	}

	user, err := s.repo.FindByUsername(ctx, input.Username)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			s.log.Warnf("login failed username=%s: not found", input.Username)
			return AuthResult{}, ErrInvalidCredentials
		}
		s.log.Errorf("login failed username=%s: %v", input.Username, err)
		return AuthResult{}, &AuthError{
			Code:    "DB_ERROR",
			Message: "failed to fetch user",
			Err:     err,
		}
	}

	if err := s.hasher.Compare(user.PasswordHash, input.Password); err != nil {
		s.log.Warnf("login failed username=%s: invalid password", input.Username)
		return AuthResult{}, ErrInvalidCredentials
	}

	accessToken, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		s.log.Errorf("login failed username=%s: token issue error: %v", input.Username, err)
		return AuthResult{}, err
	}

	s.log.Infof("login success username=%s user_id=%s", user.Username, user.ID)

	return AuthResult{
		AccessToken:      accessToken,
		RefreshToken:     refresh.RawToken,
		RefreshExpiresAt: refresh.ExpiresAt,
	}, nil
}

func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (AuthResult, error) {
	s.log.Infof("refresh token attempt")

	if refreshToken == "" {
		return AuthResult{}, ErrInvalidRefreshToken
	}

	hash := hashRefreshToken(refreshToken)

	tx, err := s.refreshTokenRepo.BeginTx(ctx)
	if err != nil {
		s.log.Errorf("refresh token failed to begin tx: %v", err)
		return AuthResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	stored, err := tx.FindByTokenHashForUpdate(ctx, hash)
	if err != nil {
		if errors.Is(err, authrepo.ErrRefreshTokenNotFound) {
			s.log.Warnf("refresh token failed: not found")
			return AuthResult{}, ErrInvalidRefreshToken
		}
		s.log.Errorf("refresh token lookup failed: %v", err)
		return AuthResult{}, err
	}

	if s.now().After(stored.ExpiresAt) {
		s.log.Warnf("refresh token expired user_id=%s", stored.UserID)
		refreshTokensExpired.Add(1)
		if err := tx.DeleteByTokenHash(ctx, hash); err != nil {
			s.log.Errorf("refresh token failed to delete expired token user_id=%s: %v", stored.UserID, err)
			return AuthResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			s.log.Errorf("refresh token failed to commit delete expired user_id=%s: %v", stored.UserID, err)
			return AuthResult{}, err
		}
		return AuthResult{}, ErrRefreshTokenExpired
	}

	user, err := s.repo.FindByID(ctx, userdomain.ID(stored.UserID))
	if err != nil {
		s.log.Errorf("refresh token failed: user lookup error user_id=%s: %v", stored.UserID, err)
		return AuthResult{}, err
	}

	if err := tx.DeleteByTokenHash(ctx, hash); err != nil {
		s.log.Errorf("refresh token failed to delete old token user_id=%s: %v", stored.UserID, err)
		return AuthResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		s.log.Errorf("refresh token failed to commit delete old token user_id=%s: %v", stored.UserID, err)
		return AuthResult{}, err
	}

	accessToken, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		s.log.Errorf("refresh token failed to issue new tokens user_id=%s: %v", stored.UserID, err)
		return AuthResult{}, err
	}

	s.log.Infof("refresh token used user_id=%s", stored.UserID)
	s.log.Infof("refresh token success user_id=%s", stored.UserID)

	refreshTokensUsed.Add(1)

	return AuthResult{
		AccessToken:      accessToken,
		RefreshToken:     refresh.RawToken,
		RefreshExpiresAt: refresh.ExpiresAt,
	}, nil
}

func (s *AuthService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	hash := hashRefreshToken(refreshToken)

	stored, err := s.refreshTokenRepo.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, authrepo.ErrRefreshTokenNotFound) {
			return nil
		}
		s.log.Errorf("revoke refresh token lookup failed: %v", err)
		return err
	}

	if err := s.refreshTokenRepo.DeleteByTokenHash(ctx, hash); err != nil {
		if errors.Is(err, authrepo.ErrRefreshTokenNotFound) {
			return nil
		}
		s.log.Errorf("revoke refresh token failed: %v", err)
		return err
	}

	s.log.Infof("refresh token revoked user_id=%s", stored.UserID)

	refreshTokensRevoked.Add(1)

	return nil
}

func (s *AuthService) RevokeAccessToken(ctx context.Context, jti string, userID string) error {
	if jti == "" {
		return nil
	}

	expiresAt := s.now().Add(15 * time.Minute)
	if err := s.revokedTokenRepo.Revoke(ctx, jti, userID, expiresAt); err != nil {
		s.log.Errorf("revoke access token failed jti=%s user_id=%s: %v", jti, userID, err)
		return err
	}

	accessTokensRevoked.Add(1)
	s.log.Infof("access token revoked jti=%s user_id=%s", jti, userID)
	return nil
}

func (s *AuthService) ParseTokenForRevoke(ctx context.Context, tokenString string) (jwtverify.Claims, error) {
	return jwtverify.ParseToken(tokenString, s.jwtSecret)
}

func (s *AuthService) issueTokens(ctx context.Context, user userdomain.User) (string, authdomain.RefreshToken, error) {
	accessToken, _, err := s.issueAccessToken(user)
	if err != nil {
		return "", authdomain.RefreshToken{}, err
	}

	refresh, err := s.issueRefreshToken(ctx, user)
	if err != nil {
		return "", authdomain.RefreshToken{}, err
	}

	return accessToken, refresh, nil
}

func (s *AuthService) issueAccessToken(user userdomain.User) (string, string, error) {
	jti, err := s.idGenerator.NewID()
	if err != nil {
		return "", "", err
	}

	expiresAt := s.now().Add(15 * time.Minute)
	claims := jwt.MapClaims{
		"sub": string(user.ID),
		"usr": user.Username,
		"jti": jti,
		"exp": expiresAt.Unix(),
		"iat": s.now().Unix(),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := t.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	accessTokensIssued.Add(1)
	return tokenString, jti, nil
}

func (s *AuthService) issueRefreshToken(ctx context.Context, user userdomain.User) (authdomain.RefreshToken, error) {
	const maxRefreshTokensPerUser = 5

	count, err := s.refreshTokenRepo.CountByUserID(ctx, string(user.ID))
	if err != nil {
		s.log.Errorf("failed to count refresh tokens user_id=%s: %v", user.ID, err)
		return authdomain.RefreshToken{}, err
	}

	if count >= maxRefreshTokensPerUser {
		if err := s.refreshTokenRepo.DeleteOldestByUserID(ctx, string(user.ID)); err != nil {
			s.log.Warnf("failed to delete oldest refresh token user_id=%s: %v", user.ID, err)
		}
	}

	rawToken, err := generateRefreshToken()
	if err != nil {
		return authdomain.RefreshToken{}, err
	}

	hash := hashRefreshToken(rawToken)

	id, err := s.idGenerator.NewID()
	if err != nil {
		return authdomain.RefreshToken{}, err
	}

	expiresAt := s.now().Add(7 * 24 * time.Hour)

	stored := authdomain.RefreshToken{
		ID:        id,
		TokenHash: hash,
		UserID:    string(user.ID),
		ExpiresAt: expiresAt,
		CreatedAt: s.now(),
	}

	if err := s.refreshTokenRepo.Create(ctx, stored); err != nil {
		return authdomain.RefreshToken{}, err
	}

	refreshTokensIssued.Add(1)

	return authdomain.RefreshToken{
		ID:        stored.ID,
		TokenHash: stored.TokenHash,
		UserID:    stored.UserID,
		ExpiresAt: stored.ExpiresAt,
		CreatedAt: stored.CreatedAt,
		RawToken:  rawToken,
	}, nil
}

func generateRefreshToken() (string, error) {
	const size = 32

	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
