package chat

import (
	"context"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	identitydomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/domain"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type mockUserRepo struct {
	findByIDFunc         func(ctx context.Context, id userdomain.ID) (userdomain.User, error)
	searchByUsernameFunc func(ctx context.Context, query string, limit int) ([]userdomain.Summary, error)
}

func (m *mockUserRepo) Create(ctx context.Context, user userdomain.User) error {
	return nil
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (userdomain.User, error) {
	return userdomain.User{}, userrepo.ErrUserNotFound
}

func (m *mockUserRepo) FindByID(ctx context.Context, id userdomain.ID) (userdomain.User, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return userdomain.User{}, userrepo.ErrUserNotFound
}

func (m *mockUserRepo) SearchByUsername(ctx context.Context, query string, limit int) ([]userdomain.Summary, error) {
	if m.searchByUsernameFunc != nil {
		return m.searchByUsernameFunc(ctx, query, limit)
	}
	return nil, nil
}

func (m *mockUserRepo) UpdateLastSeen(ctx context.Context, userID userdomain.ID) error {
	return nil
}

func (m *mockUserRepo) UpdateLastSeenBatch(ctx context.Context, userIDs []userdomain.ID) error {
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id userdomain.ID) error {
	return nil
}

type mockIdentityService struct {
	getPublicKeyFunc func(ctx context.Context, userID string) ([]byte, error)
}

func (m *mockIdentityService) CreateIdentityKey(ctx context.Context, userID string, publicKey []byte) error {
	return nil
}

func (m *mockIdentityService) UpdatePublicKey(ctx context.Context, userID string, publicKey []byte) error {
	return nil
}

func (m *mockIdentityService) GetPublicKey(ctx context.Context, userID string) ([]byte, error) {
	if m.getPublicKeyFunc != nil {
		return m.getPublicKeyFunc(ctx, userID)
	}
	return nil, commonerrors.ErrIdentityKeyNotFound
}

func (m *mockIdentityService) GetIdentityKey(ctx context.Context, userID string) (identitydomain.IdentityKey, error) {
	return identitydomain.IdentityKey{}, commonerrors.ErrIdentityKeyNotFound
}

func (m *mockIdentityService) GetFingerprint(ctx context.Context, userID string) (string, error) {
	return "", commonerrors.ErrIdentityKeyNotFound
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{}
}

func newMockIdentityService() *mockIdentityService {
	return &mockIdentityService{}
}
