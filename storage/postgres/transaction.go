package postgres

import (
	"context"
	"fmt"

	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type transactionManager struct {
	db  *pgxpool.Pool
	log logger.LoggerI
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *pgxpool.Pool, log logger.LoggerI) storage.TransactionI {
	return &transactionManager{
		db:  db,
		log: log,
	}
}

// Begin starts a new transaction
func (tm *transactionManager) Begin(ctx context.Context) (any, error) {
	tx, err := tm.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// Commit commits the transaction
func (tm *transactionManager) Commit(ctx context.Context, tx any) error {
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return fmt.Errorf("invalid transaction type")
	}

	if err := pgxTx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback rolls back the transaction
func (tm *transactionManager) Rollback(ctx context.Context, tx any) error {
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return fmt.Errorf("invalid transaction type")
	}

	if err := pgxTx.Rollback(ctx); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}
