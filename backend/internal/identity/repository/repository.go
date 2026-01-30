package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/domain"
)

type Repository interface {
	Create(ctx context.Context, key domain.IdentityKey) error
	FindByUserID(ctx context.Context, userID string) (domain.IdentityKey, error)
	Update(ctx context.Context, userID string, publicKey []byte) error
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

func (r *PgRepository) Create(ctx context.Context, key domain.IdentityKey) error {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	start := time.Now()
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO identity_keys (user_id, public_key) VALUES ($1, $2)`,
		key.UserID,
		key.PublicKey,
	)
	return db.HandleExecError(err, "create identity key", start)
}

func (r *PgRepository) FindByUserID(ctx context.Context, userID string) (domain.IdentityKey, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	start := time.Now()
	row := r.pool.QueryRow(
		ctx,
		`SELECT user_id, public_key, created_at FROM identity_keys WHERE user_id = $1`,
		userID,
	)

	var key domain.IdentityKey
	err := row.Scan(&key.UserID, &key.PublicKey, &key.CreatedAt)
	if err := db.HandleQueryError(err, commonerrors.ErrIdentityKeyNotFound, "find identity key", start); err != nil {
		return domain.IdentityKey{}, err
	}
	return key, nil
}

func (r *PgRepository) Update(ctx context.Context, userID string, publicKey []byte) error {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	start := time.Now()
	result, err := r.pool.Exec(
		ctx,
		`UPDATE identity_keys SET public_key = $2 WHERE user_id = $1`,
		userID,
		publicKey,
	)
	if err != nil {
		return db.HandleExecError(err, "update identity key", start)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrIdentityKeyNotFound
	}
	return nil
}
