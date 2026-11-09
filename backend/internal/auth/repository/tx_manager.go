package repository

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
)

type RefreshTokenTxManagerInterface interface {
	WithTx(ctx context.Context, fn func(context.Context, RefreshTokenTx) error) error
}

type RefreshTokenTxManager struct {
	beginTx func(context.Context) (pgx.Tx, error)
}

func NewRefreshTokenTxManager(pool *pgxpool.Pool) *RefreshTokenTxManager {
	return &RefreshTokenTxManager{
		beginTx: func(ctx context.Context) (pgx.Tx, error) {
			return pool.BeginTx(ctx, pgx.TxOptions{})
		},
	}
}

func (m *RefreshTokenTxManager) WithTx(ctx context.Context, fn func(context.Context, RefreshTokenTx) error) error {
	ctx, cancel := context.WithTimeout(ctx, constants.DBQueryTimeout)
	defer cancel()

	tx, err := m.beginTx(ctx)
	if err != nil {
		return err
	}

	refreshTokenTx := &pgRefreshTokenTx{tx: tx}
	defer func() {
		if p := recover(); p != nil {
			rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), constants.DBQueryTimeout)
			defer rollbackCancel()
			_ = tx.Rollback(rollbackCtx)
			panic(p)
		}
	}()

	if err := fn(ctx, refreshTokenTx); err != nil {
		rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), constants.DBQueryTimeout)
		defer rollbackCancel()
		_ = tx.Rollback(rollbackCtx)
		return err
	}

	return tx.Commit(ctx)
}
