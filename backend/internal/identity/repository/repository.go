package repository

import (
	"context"
	"errors"
	"fmt"

	pgx "github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/domain"
)

type Repository interface {
	Create(ctx context.Context, key domain.IdentityKey) error
	FindByUserID(ctx context.Context, userID string) (domain.IdentityKey, error)
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

func (r *PgRepository) Create(ctx context.Context, key domain.IdentityKey) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO identity_keys (user_id, public_key) VALUES ($1, $2)`,
		key.UserID,
		key.PublicKey,
	)
	if err != nil {
		return fmt.Errorf("failed to create identity key: %w", err)
	}
	return nil
}

func (r *PgRepository) FindByUserID(ctx context.Context, userID string) (domain.IdentityKey, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT user_id, public_key, created_at FROM identity_keys WHERE user_id = $1`,
		userID,
	)

	var key domain.IdentityKey
	err := row.Scan(&key.UserID, &key.PublicKey, &key.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.IdentityKey{}, ErrIdentityKeyNotFound
		}
		return domain.IdentityKey{}, fmt.Errorf("failed to find identity key: %w", err)
	}

	return key, nil
}

var ErrIdentityKeyNotFound = errors.New("identity key not found")
