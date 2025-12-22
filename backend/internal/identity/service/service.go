package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	identitydomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/domain"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
)

type Service interface {
	CreateIdentityKey(ctx context.Context, userID string, publicKey []byte) error
	GetPublicKey(ctx context.Context, userID string) ([]byte, error)
	GetIdentityKey(ctx context.Context, userID string) (identitydomain.IdentityKey, error)
	GetFingerprint(ctx context.Context, userID string) (string, error)
}

type IdentityService struct {
	repo identityrepo.Repository
	log  *logger.Logger
}

func NewIdentityService(repo identityrepo.Repository, log *logger.Logger) *IdentityService {
	return &IdentityService{
		repo: repo,
		log:  log,
	}
}

func (s *IdentityService) CreateIdentityKey(ctx context.Context, userID string, publicKey []byte) error {
	if len(publicKey) == 0 {
		s.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "create_identity_key_empty",
		}).Warn("create identity key failed: empty public key")
		return commonerrors.ErrInvalidPublicKey
	}

	if len(publicKey) < 50 || len(publicKey) > 200 {
		s.log.WithFields(ctx, logger.Fields{
			"user_id":    userID,
			"key_length": len(publicKey),
			"action":     "create_identity_key_invalid_length",
		}).Warnf("create identity key failed: invalid public key length %d bytes (expected SPKI format 50-200 bytes)", len(publicKey))
		return commonerrors.ErrInvalidPublicKey
	}

	key := identitydomain.IdentityKey{
		UserID:    userID,
		PublicKey: publicKey,
	}

	if err := s.repo.Create(ctx, key); err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "create_identity_key_failed",
		}).Errorf("create identity key failed: %v", err)
		return fmt.Errorf("failed to create identity key: %w", err)
	}

	s.log.WithFields(ctx, logger.Fields{
		"user_id": userID,
		"action":  "identity_key_created",
	}).Info("identity key created")
	return nil
}

func (s *IdentityService) GetPublicKey(ctx context.Context, userID string) ([]byte, error) {
	s.log.WithFields(ctx, logger.Fields{
		"user_id": userID,
		"action":  "get_identity_key_requested",
	}).Debug("identity key requested")

	key, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrIdentityKeyNotFound) {
			s.log.WithFields(ctx, logger.Fields{
				"user_id": userID,
				"action":  "get_identity_key_not_found",
			}).Warn("get identity key failed: not found")
			return nil, commonerrors.ErrIdentityKeyNotFound
		}
		s.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "get_identity_key_failed",
		}).Errorf("get identity key failed: %v", err)
		return nil, fmt.Errorf("failed to get identity key: %w", err)
	}

	return key.PublicKey, nil
}

func (s *IdentityService) GetIdentityKey(ctx context.Context, userID string) (identitydomain.IdentityKey, error) {
	key, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrIdentityKeyNotFound) {
			s.log.WithFields(ctx, logger.Fields{
				"user_id": userID,
				"action":  "get_identity_key_not_found",
			}).Warn("get identity key failed: not found")
			return identitydomain.IdentityKey{}, commonerrors.ErrIdentityKeyNotFound
		}
		s.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "get_identity_key_failed",
		}).Errorf("get identity key failed: %v", err)
		return identitydomain.IdentityKey{}, fmt.Errorf("failed to get identity key: %w", err)
	}

	return key, nil
}

func (s *IdentityService) GetFingerprint(ctx context.Context, userID string) (string, error) {
	s.log.WithFields(ctx, logger.Fields{
		"user_id": userID,
		"action":  "get_fingerprint_requested",
	}).Debug("identity fingerprint requested")

	key, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrIdentityKeyNotFound) {
			s.log.WithFields(ctx, logger.Fields{
				"user_id": userID,
				"action":  "get_fingerprint_not_found",
			}).Warn("get identity fingerprint failed: not found")
			return "", commonerrors.ErrIdentityKeyNotFound
		}
		s.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "get_fingerprint_failed",
		}).Errorf("get identity fingerprint failed: %v", err)
		return "", fmt.Errorf("failed to get identity fingerprint: %w", err)
	}

	hash := sha256.Sum256(key.PublicKey)
	fingerprint := hex.EncodeToString(hash[:])

	return fingerprint, nil
}
