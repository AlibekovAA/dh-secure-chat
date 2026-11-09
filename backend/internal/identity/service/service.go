package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	identitydomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/domain"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
)

type Service interface {
	CreateIdentityKey(ctx context.Context, userID string, publicKey []byte) error
	UpdatePublicKey(ctx context.Context, userID string, publicKey []byte) error
	GetPublicKey(ctx context.Context, userID string) ([]byte, error)
	GetFingerprint(ctx context.Context, userID string) (string, error)
}

type IdentityService struct {
	repo identityrepo.Repository
	log  *logger.Logger
}

type IdentityServiceDeps struct {
	Repo identityrepo.Repository
	Log  *logger.Logger
}

func NewIdentityService(deps IdentityServiceDeps) *IdentityService {
	return &IdentityService{
		repo: deps.Repo,
		log:  deps.Log,
	}
}

func (s *IdentityService) CreateIdentityKey(ctx context.Context, userID string, publicKey []byte) error {
	if !isValidPublicKey(publicKey) {
		s.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "create_identity_key_empty",
		}).Warn("create identity key failed: empty public key")
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
		return commonerrors.ErrIdentityKeyCreateFailed.WithCause(err)
	}

	s.log.WithFields(ctx, logger.Fields{
		"user_id": userID,
		"action":  "identity_key_created",
	}).Info("identity key created")
	return nil
}

func (s *IdentityService) UpdatePublicKey(ctx context.Context, userID string, publicKey []byte) error {
	if !isValidPublicKey(publicKey) {
		s.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "update_identity_key_empty",
		}).Warn("update identity key failed: empty public key")
		return commonerrors.ErrInvalidPublicKey
	}
	err := s.repo.Update(ctx, userID, publicKey)
	if err != nil {
		if errors.Is(err, commonerrors.ErrIdentityKeyNotFound) {
			key := identitydomain.IdentityKey{
				UserID:    userID,
				PublicKey: publicKey,
			}
			if createErr := s.repo.Create(ctx, key); createErr != nil {
				s.log.WithFields(ctx, logger.Fields{
					"user_id": userID,
					"action":  "update_identity_key_create_failed",
				}).Errorf("update identity key: create after missing row failed: %v", createErr)
				return commonerrors.ErrIdentityKeyCreateFailed.WithCause(createErr)
			}
			s.log.WithFields(ctx, logger.Fields{
				"user_id": userID,
				"action":  "identity_key_created_on_update",
			}).Info("identity key created (no existing row)")
			return nil
		}
		s.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "update_identity_key_failed",
		}).Errorf("update identity key failed: %v", err)
		return commonerrors.ErrIdentityKeyUpdateFailed.WithCause(err)
	}

	s.log.WithFields(ctx, logger.Fields{
		"user_id": userID,
		"action":  "identity_key_updated",
	}).Info("identity key updated")
	return nil
}

func isValidPublicKey(encoded []byte) bool {
	parsed, err := x509.ParsePKIXPublicKey(encoded)
	if err != nil {
		return false
	}
	publicKey, ok := parsed.(*ecdsa.PublicKey)
	return ok && publicKey.Curve == elliptic.P256()
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
		return nil, commonerrors.ErrIdentityKeyGetFailed.WithCause(err)
	}

	return key.PublicKey, nil
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
		return "", commonerrors.ErrFingerprintGetFailed.WithCause(err)
	}

	hash := sha256.Sum256(key.PublicKey)
	fingerprint := hex.EncodeToString(hash[:])

	return fingerprint, nil
}
