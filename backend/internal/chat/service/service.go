package service

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/dto"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/mapper"
	identityservice "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type ChatService struct {
	repo            userrepo.Repository
	identityService identityservice.Service
	log             *logger.Logger
}

type ChatServiceDeps struct {
	Repo            userrepo.Repository
	IdentityService identityservice.Service
	Log             *logger.Logger
}

func NewChatService(deps ChatServiceDeps) *ChatService {
	return &ChatService{
		repo:            deps.Repo,
		identityService: deps.IdentityService,
		log:             deps.Log,
	}
}

func (s *ChatService) GetMe(ctx context.Context, userID string) (dto.User, error) {
	user, err := s.repo.FindByID(ctx, userdomain.ID(userID))
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return dto.User{}, commonerrors.ErrUserNotFound.WithCause(err)
		}
		return dto.User{}, commonerrors.ErrUserGetFailed.WithCause(err)
	}
	return mapper.UserToDTO(user), nil
}

func (s *ChatService) SearchUsers(ctx context.Context, query string, limit int) ([]dto.UserSummary, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, commonerrors.ErrEmptyQuery
	}
	if len(q) > constants.MaxSearchQueryLength {
		return nil, commonerrors.NewDomainError(
			"QUERY_TOO_LONG",
			commonerrors.CategoryValidation,
			http.StatusBadRequest,
			"query is too long",
		)
	}
	if limit <= 0 {
		limit = constants.DefaultSearchLimit
	}
	if limit > constants.MaxSearchResultsLimit {
		limit = constants.MaxSearchResultsLimit
	}
	users, err := s.repo.SearchByUsername(ctx, q, limit)
	if err != nil {
		s.log.WithFields(ctx, logger.Fields{
			"query":  q,
			"limit":  limit,
			"action": "search_users_failed",
		}).Errorf("search users failed: %v", err)
		return nil, commonerrors.ErrUserSearchFailed.WithCause(err)
	}
	return mapper.UserSummariesToDTO(users), nil
}

func (s *ChatService) GetIdentityKey(ctx context.Context, userID string) ([]byte, error) {
	key, err := s.identityService.GetPublicKey(ctx, userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrIdentityKeyNotFound) {
			return nil, err
		}
		return nil, commonerrors.ErrIdentityKeyGetFailed.WithCause(err)
	}
	return key, nil
}
