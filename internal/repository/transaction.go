package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ TransactionManager = (*transactionManager)(nil)

type TransactionManager interface {
	Run(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error
}

type transactionManager struct {
	pool *pgxpool.Pool
}

func NewTransactionManager(pool *pgxpool.Pool) TransactionManager {
	return &transactionManager{pool: pool}
}

func (tm *transactionManager) Run(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			rollbackErr := tx.Rollback(ctx)
			if rollbackErr != nil {
				fmt.Printf("failed to rollback transaction in panic recovery: %v\n", rollbackErr)
			}
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %w", err, rollbackErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
