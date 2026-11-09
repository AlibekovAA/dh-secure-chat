package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v4"
)

type failingCommitTx struct {
	pgx.Tx
	commitErr  error
	rolledBack bool
}

func (tx *failingCommitTx) Commit(context.Context) error { return tx.commitErr }
func (tx *failingCommitTx) Rollback(context.Context) error {
	tx.rolledBack = true
	return nil
}

func TestWithTxReturnsCommitError(t *testing.T) {
	commitErr := errors.New("commit failed")
	tx := &failingCommitTx{commitErr: commitErr}
	manager := &RefreshTokenTxManager{beginTx: func(context.Context) (pgx.Tx, error) { return tx, nil }}

	err := manager.WithTx(context.Background(), func(context.Context, RefreshTokenTx) error { return nil })
	if !errors.Is(err, commitErr) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestWithTxRollsBackCallbackError(t *testing.T) {
	callbackErr := errors.New("callback failed")
	tx := &failingCommitTx{}
	manager := &RefreshTokenTxManager{beginTx: func(context.Context) (pgx.Tx, error) { return tx, nil }}

	err := manager.WithTx(context.Background(), func(context.Context, RefreshTokenTx) error { return callbackErr })
	if !errors.Is(err, callbackErr) || !tx.rolledBack {
		t.Fatalf("expected callback error and rollback, got err=%v rollback=%v", err, tx.rolledBack)
	}
}
