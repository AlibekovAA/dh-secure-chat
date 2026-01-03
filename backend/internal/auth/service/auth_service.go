package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	commoncrypto "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/crypto"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/resilience"
	identityservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type AuthService struct {
	repo                userrepo.Repository
	identityService     identityservice.Service
	refreshTokenRepo    authrepo.RefreshTokenRepository
	revokedTokenRepo    authrepo.RevokedTokenRepository
	hasher              commoncrypto.PasswordHasher
	idGenerator         commoncrypto.IDGenerator
	clock               clock.Clock
	log                 *logger.Logger
	dbCircuitBreaker    *resilience.CircuitBreaker
	accessTokenTTL      time.Duration
	tokenIssuer         *TokenIssuer
	refreshTokenRotator *RefreshTokenRotator
	credentialValidator *CredentialValidator
	refreshTokenCache   *RefreshTokenCache
}

type AuthServiceConfig struct {
	JWTSecret               string
	AccessTokenTTL          time.Duration
	RefreshTokenTTL         time.Duration
	MaxRefreshTokens        int
	CircuitBreakerThreshold int32
	CircuitBreakerTimeout   time.Duration
	CircuitBreakerReset     time.Duration
}

type AuthServiceDeps struct {
	Repo             userrepo.Repository
	IdentityService  identityservice.Service
	RefreshTokenRepo authrepo.RefreshTokenRepository
	RevokedTokenRepo authrepo.RevokedTokenRepository
	Hasher           commoncrypto.PasswordHasher
	IDGenerator      commoncrypto.IDGenerator
	Clock            clock.Clock
	Log              *logger.Logger
}

func NewAuthService(deps AuthServiceDeps, config AuthServiceConfig) *AuthService {
	dbCB := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Threshold:  config.CircuitBreakerThreshold,
		Timeout:    config.CircuitBreakerTimeout,
		ResetAfter: config.CircuitBreakerReset,
		Name:       "database",
		Logger:     deps.Log,
	})
	clk := deps.Clock
	if clk == nil {
		clk = clock.NewRealClock()
	}
	tokenIssuer := NewTokenIssuer(config.JWTSecret, deps.IDGenerator, config.AccessTokenTTL, clk)
	refreshTokenRotator := NewRefreshTokenRotator(deps.RefreshTokenRepo, dbCB, deps.IDGenerator, config.RefreshTokenTTL, config.MaxRefreshTokens, clk, deps.Log)
	credentialValidator := NewCredentialValidator()

	ctx := context.Background()
	refreshTokenCache := NewRefreshTokenCache(ctx, clk, deps.Log)

	return &AuthService{
		repo:                deps.Repo,
		identityService:     deps.IdentityService,
		refreshTokenRepo:    deps.RefreshTokenRepo,
		revokedTokenRepo:    deps.RevokedTokenRepo,
		hasher:              deps.Hasher,
		idGenerator:         deps.IDGenerator,
		clock:               clk,
		log:                 deps.Log,
		dbCircuitBreaker:    dbCB,
		accessTokenTTL:      config.AccessTokenTTL,
		tokenIssuer:         tokenIssuer,
		refreshTokenRotator: refreshTokenRotator,
		credentialValidator: credentialValidator,
		refreshTokenCache:   refreshTokenCache,
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

	if err := s.credentialValidator.Validate(input.Username, input.Password); err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"username": input.Username,
			"action":   "register_validation_failed",
		}).Warnf("register validation failed: %v", err)
		if validationErr, ok := AsValidationError(err); ok {
			return AuthResult{}, commonerrors.NewDomainError(
				"VALIDATION_FAILED",
				commonerrors.CategoryValidation,
				http.StatusBadRequest,
				validationErr.Error(),
			)
		}
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
		CreatedAt:    s.clock.Now(),
	}

	err = s.dbCircuitBreaker.Call(ctx, func(ctx context.Context) error {
		return s.repo.Create(ctx, user)
	})
	if err != nil {
		return s.handleDBCreateError(ctx, err, input.Username)
	}

	if len(input.IdentityPubKey) > 0 {
		if err := s.identityService.CreateIdentityKey(ctx, string(user.ID), input.IdentityPubKey); err != nil {
			if errors.Is(err, commonerrors.ErrInvalidPublicKey) {
				s.log.WithFields(ctx, logger.Fields{
					"username": input.Username,
					"user_id":  string(user.ID),
					"action":   "register_invalid_identity_key",
				}).Warn("register: invalid identity key provided, continuing without it")
			} else {
				s.log.WithFields(ctx, logger.Fields{
					"username": input.Username,
					"user_id":  string(user.ID),
					"action":   "register_identity_key_failed",
				}).Warnf("register: failed to save identity key (non-critical): %v, continuing without it", err)
			}
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

	if err := s.credentialValidator.Validate(input.Username, input.Password); err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"username": input.Username,
			"action":   "login_validation_failed",
		}).Warnf("login validation failed: %v", err)
		if validationErr, ok := AsValidationError(err); ok {
			return AuthResult{}, commonerrors.NewDomainError(
				"VALIDATION_FAILED",
				commonerrors.CategoryValidation,
				http.StatusBadRequest,
				validationErr.Error(),
			)
		}
		return AuthResult{}, err
	}

	var user userdomain.User
	err := s.dbCircuitBreaker.Call(ctx, func(ctx context.Context) error {
		var fetchErr error
		user, fetchErr = s.repo.FindByUsername(ctx, input.Username)
		return fetchErr
	})
	if err != nil {
		return s.handleDBFetchError(ctx, err, input.Username)
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

	hash := HashRefreshToken(refreshToken)

	var stored authdomain.RefreshToken
	var user userdomain.User
	var cachedUserID string
	var cacheHit bool

	if cachedToken, cachedUID, found := s.refreshTokenCache.Get(hash); found {
		if s.clock.Now().Before(cachedToken.ExpiresAt) {
			stored = cachedToken
			cachedUserID = cachedUID
			cacheHit = true
		} else {

			s.refreshTokenCache.Invalidate(hash)
		}
	}

	txMgr := s.refreshTokenRepo.TxManager()
	var err error

	if !cacheHit {
		err = s.dbCircuitBreaker.Call(ctx, func(ctx context.Context) error {
			return txMgr.WithTx(ctx, func(txCtx context.Context, tx authrepo.RefreshTokenTx) error {
				var fetchErr error
				stored, fetchErr = tx.FindByTokenHashForUpdate(txCtx, hash)
				if fetchErr != nil {
					return fetchErr
				}

				if s.clock.Now().After(stored.ExpiresAt) {
					s.log.WithFields(ctx, logger.Fields{
						"user_id": stored.UserID,
						"action":  "refresh_token_expired",
					}).Warn("refresh token expired")
					metrics.RefreshTokensExpired.Inc()
					if delErr := tx.DeleteByTokenHash(txCtx, hash); delErr != nil {
						s.log.WithFields(ctx, logger.Fields{
							"user_id": stored.UserID,
							"action":  "refresh_token_expired_delete_failed",
						}).Warnf("failed to delete expired refresh token: %v", delErr)
					}
					s.refreshTokenCache.Invalidate(hash)
					return ErrRefreshTokenExpired
				}
				s.refreshTokenCache.Set(hash, stored, stored.UserID)
				cachedUserID = stored.UserID

				var userFetchErr error
				user, userFetchErr = s.repo.FindByID(txCtx, userdomain.ID(stored.UserID))
				if userFetchErr != nil {
					return userFetchErr
				}

				if delErr := tx.DeleteByTokenHash(txCtx, hash); delErr != nil {
					return delErr
				}

				return nil
			})
		})
	} else {
		err = s.dbCircuitBreaker.Call(ctx, func(ctx context.Context) error {
			return txMgr.WithTx(ctx, func(txCtx context.Context, tx authrepo.RefreshTokenTx) error {
				var userFetchErr error
				user, userFetchErr = s.repo.FindByID(txCtx, userdomain.ID(cachedUserID))
				if userFetchErr != nil {
					return userFetchErr
				}

				if delErr := tx.DeleteByTokenHash(txCtx, hash); delErr != nil {
					return delErr
				}

				return nil
			})
		})
	}

	s.refreshTokenCache.Invalidate(hash)
	if err != nil {
		if errors.Is(err, commonerrors.ErrCircuitOpen) {
			s.log.WithFields(ctx, logger.Fields{
				"action": "refresh_token_db_circuit_open",
			}).Error("refresh token failed: database circuit breaker is open")
			return AuthResult{}, ErrServiceUnavailable.WithCause(err)
		}
		if errors.Is(err, ErrRefreshTokenExpired) {
			return AuthResult{}, err
		}
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
			"action": "refresh_token_tx_failed",
		}).Errorf("refresh token transaction failed: %v", err)
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

	metrics.RefreshTokensUsed.Inc()
	s.log.WithFields(ctx, logger.Fields{
		"user_id": stored.UserID,
		"action":  "refresh_token_success",
	}).Info("refresh token success")

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

	hash := HashRefreshToken(refreshToken)

	var stored authdomain.RefreshToken
	err := s.dbCircuitBreaker.Call(ctx, func(ctx context.Context) error {
		var fetchErr error
		stored, fetchErr = s.refreshTokenRepo.FindByTokenHash(ctx, hash)
		return fetchErr
	})
	if err != nil {
		if errors.Is(err, commonerrors.ErrCircuitOpen) {
			s.log.WithFields(ctx, logger.Fields{
				"action": "revoke_refresh_token_db_circuit_open",
			}).Error("revoke refresh token failed: database circuit breaker is open")
			return err
		}
		if errors.Is(err, authrepo.ErrRefreshTokenNotFound) {
			return nil
		}
		s.log.WithFields(ctx, logger.Fields{
			"action": "revoke_refresh_token_lookup_failed",
		}).Errorf("revoke refresh token lookup failed: %v", err)
		return err
	}

	err = s.dbCircuitBreaker.Call(ctx, func(ctx context.Context) error {
		err := s.refreshTokenRepo.DeleteByTokenHash(ctx, hash)
		if err == nil {
			s.refreshTokenCache.Invalidate(hash)
		}
		return err
	})
	if err != nil {
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

	metrics.RefreshTokensRevoked.Inc()

	return nil
}

func (s *AuthService) RevokeAccessToken(ctx context.Context, jti string, userID string) error {
	if jti == "" {
		return nil
	}

	expiresAt := s.clock.Now().Add(s.accessTokenTTL)
	err := s.dbCircuitBreaker.Call(ctx, func(ctx context.Context) error {
		return s.revokedTokenRepo.Revoke(ctx, jti, userID, expiresAt)
	})
	if err != nil {
		if errors.Is(err, commonerrors.ErrCircuitOpen) {
			s.log.WithFields(ctx, logger.Fields{
				"jti":     jti,
				"user_id": userID,
				"action":  "revoke_access_token_db_circuit_open",
			}).Error("revoke access token failed: database circuit breaker is open")
			return err
		}
		s.log.WithFields(ctx, logger.Fields{
			"jti":     jti,
			"user_id": userID,
			"action":  "revoke_access_token_failed",
		}).Errorf("revoke access token failed: %v", err)
		return err
	}

	metrics.AccessTokensRevoked.Inc()
	s.log.WithFields(ctx, logger.Fields{
		"jti":     jti,
		"user_id": userID,
		"action":  "access_token_revoked",
	}).Info("access token revoked")
	return nil
}

func (s *AuthService) ParseTokenForRevoke(ctx context.Context, tokenString string) (jwtverify.Claims, error) {
	return s.tokenIssuer.ParseToken(tokenString)
}

func (s *AuthService) handleDBCreateError(ctx context.Context, err error, username string) (AuthResult, error) {
	if errors.Is(err, commonerrors.ErrCircuitOpen) {
		s.log.WithFields(ctx, logger.Fields{
			"username": username,
			"action":   "register_db_circuit_open",
		}).Error("register failed: database circuit breaker is open")
		return AuthResult{}, ErrServiceUnavailable.WithCause(err)
	}
	if errors.Is(err, commonerrors.ErrUsernameAlreadyExists) {
		s.log.WithFields(ctx, logger.Fields{
			"username": username,
			"action":   "register_username_exists",
		}).Warn("register failed: already exists")
		return AuthResult{}, ErrUsernameTaken
	}
	s.log.WithFields(ctx, logger.Fields{
		"username": username,
		"action":   "register_create_failed",
	}).Errorf("register failed: %v", err)
	return AuthResult{}, commonerrors.NewDomainError(
		"DB_ERROR",
		commonerrors.CategoryInternal,
		http.StatusInternalServerError,
		"failed to create user",
	).WithCause(err)
}

func (s *AuthService) handleDBFetchError(ctx context.Context, err error, username string) (AuthResult, error) {
	if errors.Is(err, commonerrors.ErrCircuitOpen) {
		s.log.WithFields(ctx, logger.Fields{
			"username": username,
			"action":   "login_db_circuit_open",
		}).Error("login failed: database circuit breaker is open")
		return AuthResult{}, ErrServiceUnavailable.WithCause(err)
	}
	if errors.Is(err, userrepo.ErrUserNotFound) {
		s.log.WithFields(ctx, logger.Fields{
			"username": username,
			"action":   "login_user_not_found",
		}).Warn("login failed: not found")
		return AuthResult{}, ErrInvalidCredentials
	}
	s.log.WithFields(ctx, logger.Fields{
		"username": username,
		"action":   "login_fetch_failed",
	}).Errorf("login failed: %v", err)
	return AuthResult{}, commonerrors.NewDomainError(
		"DB_ERROR",
		commonerrors.CategoryInternal,
		http.StatusInternalServerError,
		"failed to fetch user",
	).WithCause(err)
}

func (s *AuthService) issueTokens(ctx context.Context, user userdomain.User) (string, authdomain.RefreshToken, error) {
	accessToken, _, err := s.tokenIssuer.IssueAccessToken(user)
	if err != nil {
		return "", authdomain.RefreshToken{}, err
	}

	refresh, err := s.refreshTokenRotator.IssueRefreshToken(ctx, user)
	if err != nil {
		return "", authdomain.RefreshToken{}, err
	}

	return accessToken, refresh, nil
}

func (s *AuthService) CloseRefreshTokenCache() {
	if s.refreshTokenCache != nil {
		s.refreshTokenCache.Close()
	}
}
