package repository

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
)

type RefreshTokenTxManager struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenTxManager(pool *pgxpool.Pool) *RefreshTokenTxManager {
	return &RefreshTokenTxManager{pool: pool}
}

func (m *RefreshTokenTxManager) WithTx(ctx context.Context, fn func(context.Context, RefreshTokenTx) error) error {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	refreshTokenTx := &pgRefreshTokenTx{tx: tx}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	err = fn(ctx, refreshTokenTx)
	return err
}
