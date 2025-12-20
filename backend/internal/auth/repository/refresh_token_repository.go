package repository

import (
	"context"

	pgx "github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token authdomain.RefreshToken) error
	FindByTokenHash(ctx context.Context, hash string) (authdomain.RefreshToken, error)
	DeleteByTokenHash(ctx context.Context, hash string) error
	CountByUserID(ctx context.Context, userID string) (int, error)
	DeleteOldestByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) (int64, error)
	BeginTx(ctx context.Context) (RefreshTokenTx, error)
}

type RefreshTokenTx interface {
	FindByTokenHashForUpdate(ctx context.Context, hash string) (authdomain.RefreshToken, error)
	DeleteByTokenHash(ctx context.Context, hash string) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type PgRefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPgRefreshTokenRepository(pool *pgxpool.Pool) *PgRefreshTokenRepository {
	return &PgRefreshTokenRepository{pool: pool}
}

func (r *PgRefreshTokenRepository) Create(ctx context.Context, token authdomain.RefreshToken) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO refresh_tokens (id, token_hash, user_id, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		token.ID,
		token.TokenHash,
		token.UserID,
		token.ExpiresAt,
		token.CreatedAt,
	)
	return db.HandleExecError(err, "create refresh token")
}

func (r *PgRefreshTokenRepository) FindByTokenHash(ctx context.Context, hash string) (authdomain.RefreshToken, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT id, token_hash, user_id, expires_at, created_at
		 FROM refresh_tokens
		 WHERE token_hash = $1`,
		hash,
	)

	var token authdomain.RefreshToken
	err := row.Scan(&token.ID, &token.TokenHash, &token.UserID, &token.ExpiresAt, &token.CreatedAt)
	if err := db.HandleQueryError(err, ErrRefreshTokenNotFound, "find refresh token"); err != nil {
		return authdomain.RefreshToken{}, err
	}
	return token, nil
}

func (r *PgRefreshTokenRepository) DeleteByTokenHash(ctx context.Context, hash string) error {
	_, err := r.pool.Exec(
		ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1`,
		hash,
	)
	return db.HandleExecError(err, "delete refresh token")
}

func (r *PgRefreshTokenRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1`,
		userID,
	)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, db.HandleQueryError(err, nil, "count refresh tokens")
	}
	return count, nil
}

func (r *PgRefreshTokenRepository) DeleteOldestByUserID(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(
		ctx,
		`DELETE FROM refresh_tokens
		 WHERE id = (
		 	SELECT id
		 	FROM refresh_tokens
		 	WHERE user_id = $1
		 	ORDER BY created_at ASC
		 	LIMIT 1
		 )`,
		userID,
	)
	return db.HandleExecError(err, "delete oldest refresh token")
}

func (r *PgRefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	res, err := r.pool.Exec(
		ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < NOW()`,
	)
	if err != nil {
		return 0, db.HandleExecError(err, "delete expired refresh tokens")
	}
	return res.RowsAffected(), nil
}

func (r *PgRefreshTokenRepository) BeginTx(ctx context.Context) (RefreshTokenTx, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, db.HandleExecError(err, "begin refresh token tx")
	}
	return &pgRefreshTokenTx{tx: tx}, nil
}

type pgRefreshTokenTx struct {
	tx pgx.Tx
}

func (t *pgRefreshTokenTx) FindByTokenHashForUpdate(ctx context.Context, hash string) (authdomain.RefreshToken, error) {
	row := t.tx.QueryRow(
		ctx,
		`SELECT id, token_hash, user_id, expires_at, created_at
		 FROM refresh_tokens
		 WHERE token_hash = $1
		 FOR UPDATE`,
		hash,
	)

	var token authdomain.RefreshToken
	err := row.Scan(&token.ID, &token.TokenHash, &token.UserID, &token.ExpiresAt, &token.CreatedAt)
	if err := db.HandleQueryError(err, ErrRefreshTokenNotFound, "find refresh token in tx"); err != nil {
		return authdomain.RefreshToken{}, err
	}
	return token, nil
}

func (t *pgRefreshTokenTx) DeleteByTokenHash(ctx context.Context, hash string) error {
	_, err := t.tx.Exec(
		ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1`,
		hash,
	)
	return db.HandleExecError(err, "delete refresh token in tx")
}

func (t *pgRefreshTokenTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *pgRefreshTokenTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

var ErrRefreshTokenNotFound = pgx.ErrNoRows
