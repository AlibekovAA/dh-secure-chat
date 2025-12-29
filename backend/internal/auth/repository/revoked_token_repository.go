package repository

import (
	"context"
	"errors"
	"time"

	pgx "github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
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
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	start := time.Now()
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO revoked_tokens (jti, user_id, expires_at, revoked_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (jti) DO NOTHING`,
		jti,
		userID,
		expiresAt,
	)
	return db.HandleExecError(err, "revoke token", start)
}

func (r *PgRevokedTokenRepository) IsRevoked(ctx context.Context, jti string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	start := time.Now()
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
			db.MeasureQueryDuration("check revoked token", start)
			return false, nil
		}
		return false, db.HandleQueryError(err, nil, "check revoked token", start)
	}
	db.MeasureQueryDuration("check revoked token", start)
	return exists, nil
}

func (r *PgRevokedTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	start := time.Now()
	res, err := r.pool.Exec(
		ctx,
		`DELETE FROM revoked_tokens WHERE expires_at < NOW()`,
	)
	if err != nil {
		return 0, db.HandleExecError(err, "delete expired revoked tokens", start)
	}
	db.MeasureQueryDuration("delete expired revoked tokens", start)
	return res.RowsAffected(), nil
}
