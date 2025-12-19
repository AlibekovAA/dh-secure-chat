package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	pgx "github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type RevokedTokenRepository interface {
	Revoke(ctx context.Context, jti string, userID string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
	DeleteExpired(ctx context.Context) (int64, error)
}

type PgRevokedTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPgRevokedTokenRepository(pool *pgxpool.Pool) *PgRevokedTokenRepository {
	return &PgRevokedTokenRepository{pool: pool}
}

func (r *PgRevokedTokenRepository) Revoke(ctx context.Context, jti string, userID string, expiresAt time.Time) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO revoked_tokens (jti, user_id, expires_at, revoked_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (jti) DO NOTHING`,
		jti,
		userID,
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

func (r *PgRevokedTokenRepository) IsRevoked(ctx context.Context, jti string) (bool, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM revoked_tokens
			WHERE jti = $1 AND expires_at > NOW()
		)`,
		jti,
	)

	var exists bool
	if err := row.Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check revoked token: %w", err)
	}

	return exists, nil
}

func (r *PgRevokedTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	res, err := r.pool.Exec(
		ctx,
		`DELETE FROM revoked_tokens WHERE expires_at < NOW()`,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired revoked tokens: %w", err)
	}
	return res.RowsAffected(), nil
}
