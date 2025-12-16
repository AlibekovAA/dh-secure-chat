package service

import (
	"context"
	"errors"
	"strings"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type ChatService struct {
	repo userrepo.Repository
	log  *logger.Logger
}

func NewChatService(repo userrepo.Repository, log *logger.Logger) *ChatService {
	return &ChatService{
		repo: repo,
		log:  log,
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
		return nil, errors.New("query is empty")
	}
	if limit <= 0 {
		limit = 20
	}
	users, err := s.repo.SearchByUsername(ctx, q, limit)
	if err != nil {
		s.log.Errorf("failed to search users: %v", err)
		return nil, err
	}
	return users, nil
}
