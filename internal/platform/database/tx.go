package database

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txKey struct{}
type afterCommitKey struct{}

type afterCommitHooks struct {
	fns []func()
}

// AfterCommit runs fn after the outermost transaction commits. If there is
// no transaction on ctx, fn runs immediately. Used for fire-and-forget
// WebSocket events (SD9).
func (db *DB) AfterCommit(ctx context.Context, fn func()) {
	if h, ok := ctx.Value(afterCommitKey{}).(*afterCommitHooks); ok {
		h.fns = append(h.fns, fn)
		return
	}
	fn()
}

// WithTx runs fn inside a transaction. The transaction is stored on the
// context so that any repository in any module can retrieve it via
// TxFromContext and participate in the same transaction.
//
// Nested WithTx reuses the existing transaction. Commits on nil error,
// rolls back on any error or panic. AfterCommit hooks run only after the
// outermost commit succeeds.
func (db *DB) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

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

	hooks := &afterCommitHooks{}
	ctx = context.WithValue(ctx, txKey{}, tx)
	ctx = context.WithValue(ctx, afterCommitKey{}, hooks)
	if err := fn(ctx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	for _, hook := range hooks.fns {
		hook()
	}
	return nil
}

// Querier returns the active transaction if one is on the context,
// otherwise the pool. Repositories call this, never the pool directly.
func (db *DB) Querier(ctx context.Context) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db.Pool
}
