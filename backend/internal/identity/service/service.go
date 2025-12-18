package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	identitydomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/domain"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
)

var (
	ErrIdentityKeyNotFound = errors.New("identity key not found")
	ErrInvalidPublicKey    = errors.New("invalid public key")
)

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
		s.log.Warnf("create identity key failed user_id=%s: empty public key", userID)
		return ErrInvalidPublicKey
	}

	if len(publicKey) < 50 || len(publicKey) > 200 {
		s.log.Warnf("create identity key failed user_id=%s: invalid public key length %d bytes (expected SPKI format 50-200 bytes)", userID, len(publicKey))
		return ErrInvalidPublicKey
	}

	key := identitydomain.IdentityKey{
		UserID:    userID,
		PublicKey: publicKey,
	}

	if err := s.repo.Create(ctx, key); err != nil {
		s.log.Errorf("create identity key failed user_id=%s: %v", userID, err)
		return fmt.Errorf("failed to create identity key: %w", err)
	}

	s.log.Infof("identity key created user_id=%s", userID)
	return nil
}

func (s *IdentityService) GetPublicKey(ctx context.Context, userID string) ([]byte, error) {
	s.log.Debugf("identity key requested user_id=%s", userID)

	key, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, identityrepo.ErrIdentityKeyNotFound) {
			s.log.Warnf("get identity key failed user_id=%s: not found", userID)
			return nil, ErrIdentityKeyNotFound
		}
		s.log.Errorf("get identity key failed user_id=%s: %v", userID, err)
		return nil, fmt.Errorf("failed to get identity key: %w", err)
	}

	return key.PublicKey, nil
}

func (s *IdentityService) GetIdentityKey(ctx context.Context, userID string) (identitydomain.IdentityKey, error) {
	key, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, identityrepo.ErrIdentityKeyNotFound) {
			s.log.Warnf("get identity key failed user_id=%s: not found", userID)
			return identitydomain.IdentityKey{}, ErrIdentityKeyNotFound
		}
		s.log.Errorf("get identity key failed user_id=%s: %v", userID, err)
		return identitydomain.IdentityKey{}, fmt.Errorf("failed to get identity key: %w", err)
	}

	return key, nil
}

func (s *IdentityService) GetFingerprint(ctx context.Context, userID string) (string, error) {
	s.log.Debugf("identity fingerprint requested user_id=%s", userID)

	key, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, identityrepo.ErrIdentityKeyNotFound) {
			s.log.Warnf("get identity fingerprint failed user_id=%s: not found", userID)
			return "", ErrIdentityKeyNotFound
		}
		s.log.Errorf("get identity fingerprint failed user_id=%s: %v", userID, err)
		return "", fmt.Errorf("failed to get identity fingerprint: %w", err)
	}

	hash := sha256.Sum256(key.PublicKey)
	fingerprint := hex.EncodeToString(hash[:])

	return fingerprint, nil
}
