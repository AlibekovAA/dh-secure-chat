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
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	identityservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type AuthService struct {
	repo             userrepo.Repository
	identityService  identityservice.Service
	refreshTokenRepo authrepo.RefreshTokenRepository
	revokedTokenRepo authrepo.RevokedTokenRepository
	hasher           commoncrypto.PasswordHasher
	idGenerator      commoncrypto.IDGenerator
	jwtSecret        []byte
	now              func() time.Time
	log              *logger.Logger
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	maxRefreshTokens int
}

func NewAuthService(
	repo userrepo.Repository,
	identityService identityservice.Service,
	refreshTokenRepo authrepo.RefreshTokenRepository,
	revokedTokenRepo authrepo.RevokedTokenRepository,
	hasher commoncrypto.PasswordHasher,
	idGenerator commoncrypto.IDGenerator,
	jwtSecret string,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	maxRefreshTokens int,
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
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
		maxRefreshTokens: maxRefreshTokens,
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
	s.log.WithFields(ctx, logger.Fields{
		"username": input.Username,
		"action":   "register_attempt",
	}).Info("register attempt")

	if err := validateCredentials(input.Username, input.Password); err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"username": input.Username,
			"action":   "register_validation_failed",
		}).Warnf("register validation failed: %v", err)
		return AuthResult{}, err
	}

	hash, err := s.hasher.Hash(input.Password)
	if err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"username": input.Username,
			"action":   "register_hash_failed",
		}).Errorf("register failed: password hash error: %v", err)
		return AuthResult{}, err
	}

	id, err := s.idGenerator.NewID()
	if err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"username": input.Username,
			"action":   "register_id_generation_failed",
		}).Errorf("register failed: id generation error: %v", err)
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
		if errors.Is(err, commonerrors.ErrUsernameAlreadyExists) {
			s.log.WithFields(ctx, logger.Fields{
				"username": input.Username,
				"action":   "register_username_exists",
			}).Warn("register failed: already exists")
			return AuthResult{}, ErrUsernameTaken
		}
		s.log.WithFields(ctx, logger.Fields{
			"username": input.Username,
			"action":   "register_create_failed",
		}).Errorf("register failed: %v", err)
		return AuthResult{}, &AuthError{
			Code:    "DB_ERROR",
			Message: "failed to create user",
			Err:     err,
		}
	}

	if len(input.IdentityPubKey) > 0 {
		if err := s.identityService.CreateIdentityKey(ctx, string(user.ID), input.IdentityPubKey); err != nil {
			if delErr := s.repo.Delete(ctx, user.ID); delErr != nil {
				s.log.WithFields(ctx, logger.Fields{
					"username": input.Username,
					"user_id":  string(user.ID),
					"action":   "register_delete_user_failed",
				}).Errorf("register failed: failed to delete user after identity key error: %v", delErr)
			}
			if errors.Is(err, commonerrors.ErrInvalidPublicKey) {
				return AuthResult{}, err
			}
			s.log.WithFields(ctx, logger.Fields{
				"username": input.Username,
				"user_id":  string(user.ID),
				"action":   "register_identity_key_failed",
			}).Errorf("register failed: failed to save identity key: %v", err)
			return AuthResult{}, fmt.Errorf("failed to save identity key: %w", err)
		}
	}

	accessToken, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"username": input.Username,
			"user_id":  string(user.ID),
			"action":   "register_token_issue_failed",
		}).Errorf("register failed: token issue error: %v", err)
		return AuthResult{}, err
	}

	s.log.WithFields(ctx, logger.Fields{
		"username": user.Username,
		"user_id":  string(user.ID),
		"action":   "register_success",
	}).Info("register success")

	return AuthResult{
		AccessToken:      accessToken,
		RefreshToken:     refresh.RawToken,
		RefreshExpiresAt: refresh.ExpiresAt,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	s.log.WithFields(ctx, logger.Fields{
		"username": input.Username,
		"action":   "login_attempt",
	}).Info("login attempt")

	if err := validateCredentials(input.Username, input.Password); err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"username": input.Username,
			"action":   "login_validation_failed",
		}).Warnf("login validation failed: %v", err)
		return AuthResult{}, err
	}

	user, err := s.repo.FindByUsername(ctx, input.Username)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			s.log.WithFields(ctx, logger.Fields{
				"username": input.Username,
				"action":   "login_user_not_found",
			}).Warn("login failed: not found")
			return AuthResult{}, ErrInvalidCredentials
		}
		s.log.WithFields(ctx, logger.Fields{
			"username": input.Username,
			"action":   "login_fetch_failed",
		}).Errorf("login failed: %v", err)
		return AuthResult{}, &AuthError{
			Code:    "DB_ERROR",
			Message: "failed to fetch user",
			Err:     err,
		}
	}

	if err := s.hasher.Compare(user.PasswordHash, input.Password); err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"username": input.Username,
			"action":   "login_invalid_password",
		}).Warn("login failed: invalid password")
		return AuthResult{}, ErrInvalidCredentials
	}

	accessToken, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"username": input.Username,
			"user_id":  string(user.ID),
			"action":   "login_token_issue_failed",
		}).Errorf("login failed: token issue error: %v", err)
		return AuthResult{}, err
	}

	s.log.WithFields(ctx, logger.Fields{
		"username": user.Username,
		"user_id":  string(user.ID),
		"action":   "login_success",
	}).Info("login success")

	return AuthResult{
		AccessToken:      accessToken,
		RefreshToken:     refresh.RawToken,
		RefreshExpiresAt: refresh.ExpiresAt,
	}, nil
}

func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string, clientIP string) (AuthResult, error) {
	s.log.WithFields(ctx, logger.Fields{
		"action": "refresh_token_attempt",
	}).Info("refresh token attempt")

	if refreshToken == "" {
		return AuthResult{}, ErrInvalidRefreshToken
	}

	hash := hashRefreshToken(refreshToken)

	tx, err := s.refreshTokenRepo.BeginTx(ctx)
	if err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"action": "refresh_token_begin_tx_failed",
		}).Errorf("refresh token failed to begin tx: %v", err)
		return AuthResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	stored, err := tx.FindByTokenHashForUpdate(ctx, hash)
	if err != nil {
		if errors.Is(err, authrepo.ErrRefreshTokenNotFound) {
			fields := logger.Fields{
				"action": "refresh_token_not_found",
			}
			if clientIP != "" {
				fields["client_ip"] = clientIP
			}
			s.log.WithFields(ctx, fields).Warn("refresh token failed: not found")
			return AuthResult{}, ErrInvalidRefreshToken
		}
		s.log.WithFields(ctx, logger.Fields{
			"action": "refresh_token_lookup_failed",
		}).Errorf("refresh token lookup failed: %v", err)
		return AuthResult{}, err
	}

	if s.now().After(stored.ExpiresAt) {
		s.log.WithFields(ctx, logger.Fields{
			"user_id": stored.UserID,
			"action":  "refresh_token_expired",
		}).Warn("refresh token expired")
		incrementRefreshTokensExpired()
		if err := tx.DeleteByTokenHash(ctx, hash); err != nil {
			s.log.WithFields(ctx, logger.Fields{
				"user_id": stored.UserID,
				"action":  "refresh_token_delete_expired_failed",
			}).Errorf("refresh token failed to delete expired token: %v", err)
			return AuthResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			s.log.WithFields(ctx, logger.Fields{
				"user_id": stored.UserID,
				"action":  "refresh_token_commit_delete_expired_failed",
			}).Errorf("refresh token failed to commit delete expired: %v", err)
			return AuthResult{}, err
		}
		return AuthResult{}, ErrRefreshTokenExpired
	}

	user, err := s.repo.FindByID(ctx, userdomain.ID(stored.UserID))
	if err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"user_id": stored.UserID,
			"action":  "refresh_token_user_lookup_failed",
		}).Errorf("refresh token failed: user lookup error: %v", err)
		return AuthResult{}, err
	}

	if err := tx.DeleteByTokenHash(ctx, hash); err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"user_id": stored.UserID,
			"action":  "refresh_token_delete_old_failed",
		}).Errorf("refresh token failed to delete old token: %v", err)
		return AuthResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"user_id": stored.UserID,
			"action":  "refresh_token_commit_delete_old_failed",
		}).Errorf("refresh token failed to commit delete old token: %v", err)
		return AuthResult{}, err
	}

	accessToken, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"user_id": stored.UserID,
			"action":  "refresh_token_issue_failed",
		}).Errorf("refresh token failed to issue new tokens: %v", err)
		return AuthResult{}, err
	}

	s.log.WithFields(ctx, logger.Fields{
		"user_id": stored.UserID,
		"action":  "refresh_token_used",
	}).Info("refresh token used")
	s.log.WithFields(ctx, logger.Fields{
		"user_id": stored.UserID,
		"action":  "refresh_token_success",
	}).Info("refresh token success")

	incrementRefreshTokensUsed()

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
		s.log.WithFields(ctx, logger.Fields{
			"action": "revoke_refresh_token_lookup_failed",
		}).Errorf("revoke refresh token lookup failed: %v", err)
		return err
	}

	if err := s.refreshTokenRepo.DeleteByTokenHash(ctx, hash); err != nil {
		if errors.Is(err, authrepo.ErrRefreshTokenNotFound) {
			return nil
		}
		s.log.WithFields(ctx, logger.Fields{
			"user_id": stored.UserID,
			"action":  "revoke_refresh_token_failed",
		}).Errorf("revoke refresh token failed: %v", err)
		return err
	}

	s.log.WithFields(ctx, logger.Fields{
		"user_id": stored.UserID,
		"action":  "refresh_token_revoked",
	}).Info("refresh token revoked")

	incrementRefreshTokensRevoked()

	return nil
}

func (s *AuthService) RevokeAccessToken(ctx context.Context, jti string, userID string) error {
	if jti == "" {
		return nil
	}

	expiresAt := s.now().Add(s.accessTokenTTL)
	if err := s.revokedTokenRepo.Revoke(ctx, jti, userID, expiresAt); err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"jti":     jti,
			"user_id": userID,
			"action":  "revoke_access_token_failed",
		}).Errorf("revoke access token failed: %v", err)
		return err
	}

	incrementAccessTokensRevoked()
	s.log.WithFields(ctx, logger.Fields{
		"jti":     jti,
		"user_id": userID,
		"action":  "access_token_revoked",
	}).Info("access token revoked")
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

	expiresAt := s.now().Add(s.accessTokenTTL)
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

	incrementAccessTokensIssued()
	return tokenString, jti, nil
}

func (s *AuthService) issueRefreshToken(ctx context.Context, user userdomain.User) (authdomain.RefreshToken, error) {
	count, err := s.refreshTokenRepo.CountByUserID(ctx, string(user.ID))
	if err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"user_id": string(user.ID),
			"action":  "count_refresh_tokens_failed",
		}).Errorf("failed to count refresh tokens: %v", err)
		return authdomain.RefreshToken{}, err
	}

	if count >= s.maxRefreshTokens {
		if err := s.refreshTokenRepo.DeleteOldestByUserID(ctx, string(user.ID)); err != nil {
			s.log.WithFields(ctx, logger.Fields{
				"user_id": string(user.ID),
				"action":  "delete_oldest_refresh_token_failed",
			}).Warnf("failed to delete oldest refresh token: %v", err)
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

	expiresAt := s.now().Add(s.refreshTokenTTL)

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

	incrementRefreshTokensIssued()

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
