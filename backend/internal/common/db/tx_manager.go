package db

import "context"

type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type TxManager interface {
	WithTx(ctx context.Context, fn func(context.Context, Tx) error) error
}
