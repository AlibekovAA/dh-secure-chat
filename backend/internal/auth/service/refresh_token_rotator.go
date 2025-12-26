package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	commoncrypto "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/crypto"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

type RefreshTokenRotator struct {
	refreshTokenRepo authrepo.RefreshTokenRepository
	dbCircuitBreaker *db.DBCircuitBreaker
	idGenerator      commoncrypto.IDGenerator
	clock            clock.Clock
	maxRefreshTokens int
	refreshTokenTTL  time.Duration
	log              *logger.Logger
}

func NewRefreshTokenRotator(
	refreshTokenRepo authrepo.RefreshTokenRepository,
	dbCircuitBreaker *db.DBCircuitBreaker,
	idGenerator commoncrypto.IDGenerator,
	refreshTokenTTL time.Duration,
	maxRefreshTokens int,
	clock clock.Clock,
	log *logger.Logger,
) *RefreshTokenRotator {
	return &RefreshTokenRotator{
		refreshTokenRepo: refreshTokenRepo,
		dbCircuitBreaker: dbCircuitBreaker,
		idGenerator:      idGenerator,
		clock:            clock,
		maxRefreshTokens: maxRefreshTokens,
		refreshTokenTTL:  refreshTokenTTL,
		log:              log,
	}
}

func (rtr *RefreshTokenRotator) RotateIfNeeded(ctx context.Context, userID string) error {
	var count int
	err := rtr.dbCircuitBreaker.Call(ctx, func(ctx context.Context) error {
		var countErr error
		count, countErr = rtr.refreshTokenRepo.CountByUserID(ctx, userID)
		return countErr
	})
	if err != nil {
		if errors.Is(err, commonerrors.ErrCircuitOpen) {
			rtr.log.WithFields(ctx, logger.Fields{
				"user_id": userID,
				"action":  "count_refresh_tokens_db_circuit_open",
			}).Error("failed to count refresh tokens: database circuit breaker is open")
			return err
		}
		rtr.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "count_refresh_tokens_failed",
		}).Errorf("failed to count refresh tokens: %v", err)
		return err
	}

	if count >= rtr.maxRefreshTokens {
		err = rtr.dbCircuitBreaker.Call(ctx, func(ctx context.Context) error {
			return rtr.refreshTokenRepo.DeleteOldestByUserID(ctx, userID)
		})
		if err != nil {
			rtr.log.WithFields(ctx, logger.Fields{
				"user_id": userID,
				"action":  "delete_oldest_refresh_token_failed",
			}).Warnf("failed to delete oldest refresh token: %v", err)
		}
	}

	return nil
}

func (rtr *RefreshTokenRotator) IssueRefreshToken(ctx context.Context, user userdomain.User) (authdomain.RefreshToken, error) {
	if err := rtr.RotateIfNeeded(ctx, string(user.ID)); err != nil {
		return authdomain.RefreshToken{}, err
	}

	rawToken, err := GenerateRefreshToken()
	if err != nil {
		return authdomain.RefreshToken{}, err
	}

	hash := HashRefreshToken(rawToken)

	id, err := rtr.idGenerator.NewID()
	if err != nil {
		return authdomain.RefreshToken{}, err
	}

	expiresAt := rtr.clock.Now().Add(rtr.refreshTokenTTL)

	stored := authdomain.RefreshToken{
		ID:        id,
		TokenHash: hash,
		UserID:    string(user.ID),
		ExpiresAt: expiresAt,
		CreatedAt: rtr.clock.Now(),
	}

	err = rtr.dbCircuitBreaker.Call(ctx, func(ctx context.Context) error {
		return rtr.refreshTokenRepo.Create(ctx, stored)
	})
	if err != nil {
		if errors.Is(err, commonerrors.ErrCircuitOpen) {
			rtr.log.WithFields(ctx, logger.Fields{
				"user_id": string(user.ID),
				"action":  "create_refresh_token_db_circuit_open",
			}).Error("failed to create refresh token: database circuit breaker is open")
		}
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

func GenerateRefreshToken() (string, error) {
	const size = 32

	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
