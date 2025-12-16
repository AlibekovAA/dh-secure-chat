package service

import (
	"context"
	"errors"
	"strings"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

var ErrEmptyQuery = errors.New("query is empty")

type ChatService struct {
	repo         userrepo.Repository
	identityRepo identityrepo.Repository
	log          *logger.Logger
}

func NewChatService(repo userrepo.Repository, identityRepo identityrepo.Repository, log *logger.Logger) *ChatService {
	return &ChatService{
		repo:         repo,
		identityRepo: identityRepo,
		log:          log,
	}
}

func (s *ChatService) GetMe(ctx context.Context, userID string) (userdomain.User, error) {
	user, err := s.repo.FindByID(ctx, userdomain.ID(userID))
	if err != nil {
		return userdomain.User{}, err
	}
	return user, nil
}

func (s *ChatService) SearchUsers(ctx context.Context, query string, limit int) ([]userdomain.Summary, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, ErrEmptyQuery
	}
	if limit <= 0 {
		limit = 20
	}
	users, err := s.repo.SearchByUsername(ctx, q, limit)
	if err != nil {
		s.log.Errorf("search users failed query=%q limit=%d: %v", q, limit, err)
		return nil, err
	}
	return users, nil
}

func (s *ChatService) GetIdentityKey(ctx context.Context, userID string) ([]byte, error) {
	key, err := s.identityRepo.FindByUserID(ctx, userID)
	if err != nil {
		s.log.Errorf("get identity key failed user_id=%s: %v", userID, err)
		return nil, err
	}
	return key.PublicKey, nil
}
