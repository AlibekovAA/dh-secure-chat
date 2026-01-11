package auth

import (
	"context"
	"time"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	identitydomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/domain"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type mockUserRepo struct {
	createFunc              func(ctx context.Context, user userdomain.User) error
	findByUsernameFunc      func(ctx context.Context, username string) (userdomain.User, error)
	findByIDFunc            func(ctx context.Context, id userdomain.ID) (userdomain.User, error)
	searchByUsernameFunc    func(ctx context.Context, query string, limit int) ([]userdomain.Summary, error)
	updateLastSeenFunc      func(ctx context.Context, userID userdomain.ID) error
	updateLastSeenBatchFunc func(ctx context.Context, userIDs []userdomain.ID) error
	deleteFunc              func(ctx context.Context, id userdomain.ID) error
}

func (m *mockUserRepo) Create(ctx context.Context, user userdomain.User) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, user)
	}
	return nil
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (userdomain.User, error) {
	if m.findByUsernameFunc != nil {
		return m.findByUsernameFunc(ctx, username)
	}
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
	if m.updateLastSeenFunc != nil {
		return m.updateLastSeenFunc(ctx, userID)
	}
	return nil
}

func (m *mockUserRepo) UpdateLastSeenBatch(ctx context.Context, userIDs []userdomain.ID) error {
	if m.updateLastSeenBatchFunc != nil {
		return m.updateLastSeenBatchFunc(ctx, userIDs)
	}
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id userdomain.ID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

type mockIdentityService struct {
	createIdentityKeyFunc func(ctx context.Context, userID string, publicKey []byte) error
	getPublicKeyFunc      func(ctx context.Context, userID string) ([]byte, error)
	getIdentityKeyFunc    func(ctx context.Context, userID string) (identitydomain.IdentityKey, error)
	getFingerprintFunc    func(ctx context.Context, userID string) (string, error)
}

func (m *mockIdentityService) CreateIdentityKey(ctx context.Context, userID string, publicKey []byte) error {
	if m.createIdentityKeyFunc != nil {
		return m.createIdentityKeyFunc(ctx, userID, publicKey)
	}
	return nil
}

func (m *mockIdentityService) GetPublicKey(ctx context.Context, userID string) ([]byte, error) {
	if m.getPublicKeyFunc != nil {
		return m.getPublicKeyFunc(ctx, userID)
	}
	return nil, commonerrors.ErrIdentityKeyNotFound
}

func (m *mockIdentityService) GetIdentityKey(ctx context.Context, userID string) (identitydomain.IdentityKey, error) {
	if m.getIdentityKeyFunc != nil {
		return m.getIdentityKeyFunc(ctx, userID)
	}
	return identitydomain.IdentityKey{}, commonerrors.ErrIdentityKeyNotFound
}

func (m *mockIdentityService) GetFingerprint(ctx context.Context, userID string) (string, error) {
	if m.getFingerprintFunc != nil {
		return m.getFingerprintFunc(ctx, userID)
	}
	return "", commonerrors.ErrIdentityKeyNotFound
}

type mockRefreshTokenRepo struct {
	createFunc               func(ctx context.Context, token authdomain.RefreshToken) error
	findByTokenHashFunc      func(ctx context.Context, hash string) (authdomain.RefreshToken, error)
	deleteByTokenHashFunc    func(ctx context.Context, hash string) error
	deleteExcessByUserIDFunc func(ctx context.Context, userID string, maxTokens int) error
	deleteExpiredFunc        func(ctx context.Context) (int64, error)
	txManagerFunc            func() authrepo.RefreshTokenTxManagerInterface
}

func (m *mockRefreshTokenRepo) Create(ctx context.Context, token authdomain.RefreshToken) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, token)
	}
	return nil
}

func (m *mockRefreshTokenRepo) FindByTokenHash(ctx context.Context, hash string) (authdomain.RefreshToken, error) {
	if m.findByTokenHashFunc != nil {
		return m.findByTokenHashFunc(ctx, hash)
	}
	return authdomain.RefreshToken{}, authrepo.ErrRefreshTokenNotFound
}

func (m *mockRefreshTokenRepo) DeleteByTokenHash(ctx context.Context, hash string) error {
	if m.deleteByTokenHashFunc != nil {
		return m.deleteByTokenHashFunc(ctx, hash)
	}
	return nil
}

func (m *mockRefreshTokenRepo) DeleteExcessByUserID(ctx context.Context, userID string, maxTokens int) error {
	if m.deleteExcessByUserIDFunc != nil {
		return m.deleteExcessByUserIDFunc(ctx, userID, maxTokens)
	}
	return nil
}

func (m *mockRefreshTokenRepo) DeleteExpired(ctx context.Context) (int64, error) {
	if m.deleteExpiredFunc != nil {
		return m.deleteExpiredFunc(ctx)
	}
	return 0, nil
}

type testRefreshTokenTxManager struct {
	withTxFunc func(ctx context.Context, fn func(context.Context, authrepo.RefreshTokenTx) error) error
}

func (m *testRefreshTokenTxManager) WithTx(ctx context.Context, fn func(context.Context, authrepo.RefreshTokenTx) error) error {
	if m.withTxFunc != nil {
		return m.withTxFunc(ctx, fn)
	}
	mockTx := &mockRefreshTokenTx{}
	return fn(ctx, mockTx)
}

func newTestRefreshTokenTxManagerWithFunc(withTxFunc func(ctx context.Context, fn func(context.Context, authrepo.RefreshTokenTx) error) error) authrepo.RefreshTokenTxManagerInterface {
	return &testRefreshTokenTxManager{
		withTxFunc: withTxFunc,
	}
}

func (m *mockRefreshTokenRepo) TxManager() authrepo.RefreshTokenTxManagerInterface {
	if m.txManagerFunc != nil {
		return m.txManagerFunc()
	}
	return nil
}

type mockRefreshTokenTx struct {
	findByTokenHashForUpdateFunc func(ctx context.Context, hash string) (authdomain.RefreshToken, error)
	deleteByTokenHashFunc        func(ctx context.Context, hash string) error
	commitFunc                   func(ctx context.Context) error
	rollbackFunc                 func(ctx context.Context) error
}

func (m *mockRefreshTokenTx) FindByTokenHashForUpdate(ctx context.Context, hash string) (authdomain.RefreshToken, error) {
	if m.findByTokenHashForUpdateFunc != nil {
		return m.findByTokenHashForUpdateFunc(ctx, hash)
	}
	return authdomain.RefreshToken{}, authrepo.ErrRefreshTokenNotFound
}

func (m *mockRefreshTokenTx) DeleteByTokenHash(ctx context.Context, hash string) error {
	if m.deleteByTokenHashFunc != nil {
		return m.deleteByTokenHashFunc(ctx, hash)
	}
	return nil
}

func (m *mockRefreshTokenTx) Commit(ctx context.Context) error {
	if m.commitFunc != nil {
		return m.commitFunc(ctx)
	}
	return nil
}

func (m *mockRefreshTokenTx) Rollback(ctx context.Context) error {
	if m.rollbackFunc != nil {
		return m.rollbackFunc(ctx)
	}
	return nil
}

type mockRevokedTokenRepo struct {
	revokeFunc        func(ctx context.Context, jti string, userID string, expiresAt time.Time) error
	isRevokedFunc     func(ctx context.Context, jti string) (bool, error)
	deleteExpiredFunc func(ctx context.Context) (int64, error)
}

func (m *mockRevokedTokenRepo) Revoke(ctx context.Context, jti string, userID string, expiresAt time.Time) error {
	if m.revokeFunc != nil {
		return m.revokeFunc(ctx, jti, userID, expiresAt)
	}
	return nil
}

func (m *mockRevokedTokenRepo) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if m.isRevokedFunc != nil {
		return m.isRevokedFunc(ctx, jti)
	}
	return false, nil
}

func (m *mockRevokedTokenRepo) DeleteExpired(ctx context.Context) (int64, error) {
	if m.deleteExpiredFunc != nil {
		return m.deleteExpiredFunc(ctx)
	}
	return 0, nil
}

type mockHasher struct {
	hashFunc    func(password string) (string, error)
	compareFunc func(hash string, password string) error
}

func (m *mockHasher) Hash(password string) (string, error) {
	if m.hashFunc != nil {
		return m.hashFunc(password)
	}
	return "hashed_" + password, nil
}

func (m *mockHasher) Compare(hash string, password string) error {
	if m.compareFunc != nil {
		return m.compareFunc(hash, password)
	}
	return nil
}

type mockIDGenerator struct {
	newIDFunc func() (string, error)
}

func (m *mockIDGenerator) NewID() (string, error) {
	if m.newIDFunc != nil {
		return m.newIDFunc()
	}
	return "test-id-123", nil
}
