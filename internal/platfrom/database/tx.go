package database

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txKey struct{}

// WithTx runs fn inside a transaction. The transaction is stored on the
// context so that any repository in any module can retrieve it via
// TxFromContext and participate in the same transaction.
//
// Commits on nil error, rolls back on any error or panic.
func (db *DB) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	ctx = context.WithValue(ctx, txKey{}, tx)
	if err := fn(ctx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// Querier returns the active transaction if one is on the context,
// otherwise the pool. Repositories call this, never the pool directly.
func (db *DB) Querier(ctx context.Context) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db.Pool
}
