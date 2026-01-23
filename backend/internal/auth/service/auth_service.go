package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
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

type Service interface {
	Register(ctx context.Context, input RegisterInput) (AuthResult, error)
	Login(ctx context.Context, input LoginInput) (AuthResult, error)
	RefreshAccessToken(ctx context.Context, refreshToken string, clientIP string) (AuthResult, error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
	RevokeAccessToken(ctx context.Context, jti string, userID string) error
	ParseTokenForRevoke(ctx context.Context, tokenString string) (jwtverify.Claims, error)
	CloseRefreshTokenCache()
}

type AuthService struct {
	repo                userrepo.Repository
	identityService     identityservice.Service
	refreshTokenRepo    authrepo.RefreshTokenRepository
	revokedTokenRepo    authrepo.RevokedTokenRepository
	hasher              commoncrypto.PasswordHasher
	idGenerator         commoncrypto.IDGenerator
	clock               clock.Clock
	log                 *logger.Logger
	dbCircuitBreaker    resilience.CircuitBreakerInterface
	accessTokenTTL      time.Duration
	tokenIssuer         TokenIssuerInterface
	refreshTokenRotator RefreshTokenRotatorInterface
	credentialValidator CredentialValidator
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
	databaseCircuitBreaker := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Threshold:  config.CircuitBreakerThreshold,
		Timeout:    config.CircuitBreakerTimeout,
		ResetAfter: config.CircuitBreakerReset,
		Name:       constants.CircuitBreakerDatabaseName,
		Logger:     deps.Log,
	})
	timeClock := deps.Clock
	if timeClock == nil {
		timeClock = clock.NewRealClock()
	}
	tokenIssuer := NewTokenIssuer(config.JWTSecret, deps.IDGenerator, config.AccessTokenTTL, timeClock)
	refreshTokenRotator := NewRefreshTokenRotator(deps.RefreshTokenRepo, databaseCircuitBreaker, deps.IDGenerator, config.RefreshTokenTTL, config.MaxRefreshTokens, timeClock, deps.Log)
	credentialValidator := NewCredentialValidator()

	ctx := context.Background()
	refreshTokenCache := NewRefreshTokenCache(ctx, timeClock, deps.Log)

	return &AuthService{
		repo:                deps.Repo,
		identityService:     deps.IdentityService,
		refreshTokenRepo:    deps.RefreshTokenRepo,
		revokedTokenRepo:    deps.RevokedTokenRepo,
		hasher:              deps.Hasher,
		idGenerator:         deps.IDGenerator,
		clock:               timeClock,
		log:                 deps.Log,
		dbCircuitBreaker:    databaseCircuitBreaker,
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
		return AuthResult{}, commonerrors.NewDomainError(
			"PASSWORD_HASH_FAILED",
			commonerrors.CategoryInternal,
			http.StatusInternalServerError,
			"failed to hash password",
		).WithCause(err)
	}

	id, err := s.idGenerator.NewID()
	if err != nil {
		return AuthResult{}, commonerrors.NewDomainError(
			"ID_GENERATION_FAILED",
			commonerrors.CategoryInternal,
			http.StatusInternalServerError,
			"failed to generate user id",
		).WithCause(err)
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
		return s.handleDBError(ctx, err, input.Username, dbErrorConfig{
			operation:             "register",
			specificError:         commonerrors.ErrUsernameAlreadyExists,
			specificErrorResponse: ErrUsernameTaken,
			errorMessage:          "failed to create user",
		})
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
		return AuthResult{}, commonerrors.NewDomainError(
			"TOKEN_ISSUE_FAILED",
			commonerrors.CategoryInternal,
			http.StatusInternalServerError,
			"failed to issue tokens",
		).WithCause(err)
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
		return s.handleDBError(ctx, err, input.Username, dbErrorConfig{
			operation:             "login",
			specificError:         userrepo.ErrUserNotFound,
			specificErrorResponse: ErrInvalidCredentials,
			errorMessage:          "failed to fetch user",
		})
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
		return AuthResult{}, commonerrors.NewDomainError(
			"TOKEN_ISSUE_FAILED",
			commonerrors.CategoryInternal,
			http.StatusInternalServerError,
			"failed to issue tokens",
		).WithCause(err)
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
		stored = cachedToken
		cachedUserID = cachedUID
		cacheHit = true
	}

	txMgr := s.refreshTokenRepo.TxManager()
	var err error

	err = s.dbCircuitBreaker.Call(ctx, func(ctx context.Context) error {
		return txMgr.WithTx(ctx, func(txCtx context.Context, tx authrepo.RefreshTokenTx) error {
			if !cacheHit {
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
			}

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

	s.refreshTokenCache.Invalidate(hash)
	if err != nil {
		if errors.Is(err, commonerrors.ErrCircuitOpen) {
			s.log.WithFields(ctx, logger.Fields{
				"action": "refresh_token_db_circuit_open",
			}).Error("refresh token failed: database circuit breaker is open")
			return AuthResult{}, ErrServiceUnavailable.WithCause(err)
		}
		if errors.Is(err, ErrRefreshTokenExpired) {
			return AuthResult{}, ErrRefreshTokenExpired
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
		return AuthResult{}, commonerrors.NewDomainError(
			"REFRESH_TOKEN_TX_FAILED",
			commonerrors.CategoryInternal,
			http.StatusInternalServerError,
			"failed to process refresh token transaction",
		).WithCause(err)
	}

	accessToken, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		return AuthResult{}, commonerrors.NewDomainError(
			"TOKEN_ISSUE_FAILED",
			commonerrors.CategoryInternal,
			http.StatusInternalServerError,
			"failed to issue tokens",
		).WithCause(err)
	}

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
			return ErrServiceUnavailable.WithCause(err)
		}
		if errors.Is(err, authrepo.ErrRefreshTokenNotFound) {
			return nil
		}
		return commonerrors.NewDomainError(
			"REVOKE_REFRESH_TOKEN_LOOKUP_FAILED",
			commonerrors.CategoryInternal,
			http.StatusInternalServerError,
			"failed to lookup refresh token for revocation",
		).WithCause(err)
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
		return commonerrors.NewDomainError(
			"REVOKE_REFRESH_TOKEN_FAILED",
			commonerrors.CategoryInternal,
			http.StatusInternalServerError,
			"failed to revoke refresh token",
		).WithCause(err)
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
			return ErrServiceUnavailable.WithCause(err)
		}
		return commonerrors.NewDomainError(
			"REVOKE_ACCESS_TOKEN_FAILED",
			commonerrors.CategoryInternal,
			http.StatusInternalServerError,
			"failed to revoke access token",
		).WithCause(err)
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

type dbErrorConfig struct {
	operation             string
	specificError         error
	specificErrorResponse commonerrors.DomainError
	errorMessage          string
}

func (s *AuthService) handleDBError(ctx context.Context, err error, username string, config dbErrorConfig) (AuthResult, error) {
	if errors.Is(err, commonerrors.ErrCircuitOpen) {
		s.log.WithFields(ctx, logger.Fields{
			"username": username,
			"action":   config.operation + "_db_circuit_open",
		}).Errorf("%s failed: database circuit breaker is open", config.operation)
		return AuthResult{}, ErrServiceUnavailable.WithCause(err)
	}
	if errors.Is(err, config.specificError) {
		actionField := config.operation + "_" + getSpecificErrorAction(config.specificError)
		s.log.WithFields(ctx, logger.Fields{
			"username": username,
			"action":   actionField,
		}).Warnf("%s failed: %v", config.operation, err)
		return AuthResult{}, config.specificErrorResponse
	}
	s.log.WithFields(ctx, logger.Fields{
		"username": username,
		"action":   config.operation + "_db_failed",
	}).Errorf("%s failed: %v", config.operation, err)
	return AuthResult{}, commonerrors.NewDomainError(
		"DB_ERROR",
		commonerrors.CategoryInternal,
		http.StatusInternalServerError,
		config.errorMessage,
	).WithCause(err)
}

func getSpecificErrorAction(err error) string {
	if errors.Is(err, commonerrors.ErrUsernameAlreadyExists) {
		return "username_exists"
	}
	if errors.Is(err, userrepo.ErrUserNotFound) {
		return "user_not_found"
	}
	return "specific_error"
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
