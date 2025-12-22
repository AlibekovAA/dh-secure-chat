package service

import (
	"context"
	"fmt"
	"strings"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	identityservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type ChatService struct {
	repo            userrepo.Repository
	identityService identityservice.Service
	log             *logger.Logger
}

func NewChatService(repo userrepo.Repository, identityService identityservice.Service, log *logger.Logger) *ChatService {
	return &ChatService{
		repo:            repo,
		identityService: identityService,
		log:             log,
	}
}

func (s *ChatService) GetMe(ctx context.Context, userID string) (userdomain.User, error) {
	user, err := s.repo.FindByID(ctx, userdomain.ID(userID))
	if err != nil {
		return userdomain.User{}, fmt.Errorf("get user by id %s: %w", userID, err)
	}
	return user, nil
}

func (s *ChatService) SearchUsers(ctx context.Context, query string, limit int) ([]userdomain.Summary, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, commonerrors.ErrEmptyQuery
	}
	if limit <= 0 {
		limit = 20
	}
	users, err := s.repo.SearchByUsername(ctx, q, limit)
	if err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"query":  q,
			"limit":  limit,
			"action": "search_users_failed",
		}).Errorf("search users failed: %v", err)
		return nil, fmt.Errorf("search users query=%q limit=%d: %w", q, limit, err)
	}
	return users, nil
}

func (s *ChatService) GetIdentityKey(ctx context.Context, userID string) ([]byte, error) {
	key, err := s.identityService.GetPublicKey(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get identity key for user %s: %w", userID, err)
	}
	return key, nil
}
