package repository

import (
	"context"
	"time"

	pgx "github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	authdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/domain"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token authdomain.RefreshToken) error
	FindByTokenHash(ctx context.Context, hash string) (authdomain.RefreshToken, error)
	DeleteByTokenHash(ctx context.Context, hash string) error
	DeleteExcessByUserID(ctx context.Context, userID string, maxTokens int) error
	DeleteExpired(ctx context.Context) (int64, error)
	TxManager() RefreshTokenTxManagerInterface
}

type RefreshTokenTx interface {
	FindByTokenHashWithUserForUpdate(ctx context.Context, hash string) (authdomain.RefreshToken, userdomain.User, error)
	DeleteByTokenHash(ctx context.Context, hash string) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type PgRefreshTokenRepository struct {
	pool  *pgxpool.Pool
	txMgr *RefreshTokenTxManager
}

func NewPgRefreshTokenRepository(pool *pgxpool.Pool) *PgRefreshTokenRepository {
	return &PgRefreshTokenRepository{
		pool:  pool,
		txMgr: NewRefreshTokenTxManager(pool),
	}
}

func (r *PgRefreshTokenRepository) TxManager() RefreshTokenTxManagerInterface {
	return r.txMgr
}

func (r *PgRefreshTokenRepository) Create(ctx context.Context, token authdomain.RefreshToken) error {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	start := time.Now()
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
	return db.HandleExecError(err, "create refresh token", start)
}

func (r *PgRefreshTokenRepository) FindByTokenHash(ctx context.Context, hash string) (authdomain.RefreshToken, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	start := time.Now()
	row := r.pool.QueryRow(
		ctx,
		`SELECT id, token_hash, user_id, expires_at, created_at
		 FROM refresh_tokens
		 WHERE token_hash = $1`,
		hash,
	)

	var token authdomain.RefreshToken
	err := row.Scan(&token.ID, &token.TokenHash, &token.UserID, &token.ExpiresAt, &token.CreatedAt)
	if err := db.HandleQueryError(err, ErrRefreshTokenNotFound, "find refresh token", start); err != nil {
		return authdomain.RefreshToken{}, err
	}
	return token, nil
}

func (r *PgRefreshTokenRepository) DeleteByTokenHash(ctx context.Context, hash string) error {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	start := time.Now()
	_, err := r.pool.Exec(
		ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1`,
		hash,
	)
	return db.HandleExecError(err, "delete refresh token", start)
}

func (r *PgRefreshTokenRepository) DeleteExcessByUserID(ctx context.Context, userID string, maxTokens int) error {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	start := time.Now()
	_, err := r.pool.Exec(
		ctx,
		`DELETE FROM refresh_tokens
		 WHERE user_id = $1
		 AND id NOT IN (
		 	SELECT id
		 	FROM refresh_tokens
		 	WHERE user_id = $1
		 	ORDER BY created_at DESC
		 	LIMIT $2
		 )`,
		userID,
		maxTokens,
	)
	return db.HandleExecError(err, "delete excess refresh tokens", start)
}

func (r *PgRefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	start := time.Now()
	res, err := r.pool.Exec(
		ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < NOW()`,
	)
	if err != nil {
		return 0, db.HandleExecError(err, "delete expired refresh tokens", start)
	}
	db.MeasureQueryDuration("delete expired refresh tokens", start)
	return res.RowsAffected(), nil
}

type pgRefreshTokenTx struct {
	tx pgx.Tx
}

func (t *pgRefreshTokenTx) FindByTokenHashWithUserForUpdate(ctx context.Context, hash string) (authdomain.RefreshToken, userdomain.User, error) {
	start := time.Now()
	row := t.tx.QueryRow(
		ctx,
		`SELECT rt.id, rt.token_hash, rt.user_id, rt.expires_at, rt.created_at,
		        u.id, u.username, u.password_hash, u.created_at, u.last_seen_at
		 FROM refresh_tokens rt
		 INNER JOIN users u ON rt.user_id = u.id
		 WHERE rt.token_hash = $1
		 FOR UPDATE OF rt`,
		hash,
	)

	var token authdomain.RefreshToken
	var user userdomain.User
	err := row.Scan(
		&token.ID, &token.TokenHash, &token.UserID, &token.ExpiresAt, &token.CreatedAt,
		&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.LastSeenAt,
	)
	if err := db.HandleQueryError(err, ErrRefreshTokenNotFound, "find refresh token with user in tx", start); err != nil {
		return authdomain.RefreshToken{}, userdomain.User{}, err
	}
	return token, user, nil
}

func (t *pgRefreshTokenTx) DeleteByTokenHash(ctx context.Context, hash string) error {
	start := time.Now()
	_, err := t.tx.Exec(
		ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1`,
		hash,
	)
	return db.HandleExecError(err, "delete refresh token in tx", start)
}

func (t *pgRefreshTokenTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *pgRefreshTokenTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

var ErrRefreshTokenNotFound = pgx.ErrNoRows
