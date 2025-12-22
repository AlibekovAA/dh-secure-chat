package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgconn"
	pgx "github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/db"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

type Repository interface {
	Create(ctx context.Context, user domain.User) error
	FindByUsername(ctx context.Context, username string) (domain.User, error)
	FindByID(ctx context.Context, id domain.ID) (domain.User, error)
	SearchByUsername(ctx context.Context, query string, limit int) ([]domain.Summary, error)
	UpdateLastSeen(ctx context.Context, userID domain.ID) error
	Delete(ctx context.Context, id domain.ID) error
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

func (r *PgRepository) Create(ctx context.Context, user domain.User) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO users (id, username, password_hash) VALUES ($1, $2, $3)`,
		string(user.ID),
		user.Username,
		user.PasswordHash,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return commonerrors.ErrUsernameAlreadyExists
		}
		return db.HandleExecError(err, "create user")
	}
	return nil
}

func (r *PgRepository) FindByUsername(ctx context.Context, username string) (domain.User, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT id, username, password_hash, created_at, last_seen_at FROM users WHERE username = $1`,
		username,
	)

	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.LastSeenAt)
	if err := db.HandleQueryError(err, ErrUserNotFound, "find user by username"); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (r *PgRepository) FindByID(ctx context.Context, id domain.ID) (domain.User, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT id, username, password_hash, created_at, last_seen_at FROM users WHERE id = $1`,
		string(id),
	)

	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.LastSeenAt)
	if err := db.HandleQueryError(err, ErrUserNotFound, "find user by id"); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (r *PgRepository) SearchByUsername(ctx context.Context, query string, limit int) ([]domain.Summary, error) {
	searchPattern := "%" + query + "%"
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, username, created_at
		 FROM users
		 WHERE username ILIKE $1
		 ORDER BY username ASC
		 LIMIT $2`,
		searchPattern,
		limit,
	)
	if err != nil {
		return nil, db.HandleExecError(err, "search users")
	}
	defer rows.Close()

	users := make([]domain.Summary, 0, limit)
	for rows.Next() {
		var u domain.Summary
		if err := rows.Scan(&u.ID, &u.Username, &u.CreatedAt); err != nil {
			return nil, db.HandleQueryError(err, nil, "scan user")
		}
		users = append(users, u)
	}

	if rows.Err() != nil {
		return nil, db.HandleQueryError(rows.Err(), nil, "iterate rows")
	}

	return users, nil
}

func (r *PgRepository) UpdateLastSeen(ctx context.Context, userID domain.ID) error {
	_, err := r.pool.Exec(
		ctx,
		`UPDATE users SET last_seen_at = NOW() WHERE id = $1`,
		string(userID),
	)
	return db.HandleExecError(err, "update last_seen_at")
}

func (r *PgRepository) Delete(ctx context.Context, id domain.ID) error {
	_, err := r.pool.Exec(
		ctx,
		`DELETE FROM users WHERE id = $1`,
		string(id),
	)
	return db.HandleExecError(err, "delete user")
}

var ErrUserNotFound = pgx.ErrNoRows
