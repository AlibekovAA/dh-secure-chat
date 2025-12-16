package service

import (
	"context"

	identitydomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/domain"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
)

type identityRepoAdapter struct {
	repo identityrepo.Repository
}

func NewIdentityRepoAdapter(repo identityrepo.Repository) IdentityRepo {
	return &identityRepoAdapter{repo: repo}
}

func (a *identityRepoAdapter) Create(ctx context.Context, userID string, publicKey []byte) error {
	key := identitydomain.IdentityKey{
		UserID:    userID,
		PublicKey: publicKey,
	}
	return a.repo.Create(ctx, key)
}
