package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgconn"
	pgx "github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

type Repository interface {
	Create(ctx context.Context, user domain.User) error
	FindByUsername(ctx context.Context, username string) (domain.User, error)
	FindByID(ctx context.Context, id domain.ID) (domain.User, error)
	SearchByUsername(ctx context.Context, query string, limit int) ([]domain.Summary, error)
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
			return ErrUsernameAlreadyExists
		}
		return err
	}
	return nil
}

func (r *PgRepository) FindByUsername(ctx context.Context, username string) (domain.User, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE username = $1`,
		username,
	)

	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func (r *PgRepository) FindByID(ctx context.Context, id domain.ID) (domain.User, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE id = $1`,
		string(id),
	)

	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func (r *PgRepository) SearchByUsername(ctx context.Context, query string, limit int) ([]domain.Summary, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, username, created_at
		 FROM users
		 WHERE username ILIKE '%' || $1 || '%'
		 ORDER BY username ASC
		 LIMIT $2`,
		query,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.Summary
	for rows.Next() {
		var u domain.Summary
		if err := rows.Scan(&u.ID, &u.Username, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return users, nil
}

var ErrUserNotFound = pgx.ErrNoRows

var ErrUsernameAlreadyExists = errors.New("username already exists")
