package database

import (
	"context"

	"gorm.io/gorm"
)

// TxManager handles database transactions via context
type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type txManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) TxManager {
	return &txManager{db: db}
}

func (m *txManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	// If transaction already started, reuse it?
	// For simplicity, we assume nested transactions use savepoints or just reuse the parent.
	// GORM supports nested transactions automatically if using the same *gorm.DB instance inside a transaction.
	// But here we might be creating a new one.
	
	// Check if tx is already in context
	if _, ok := ctx.Value(TxKey).(*gorm.DB); ok {
		return fn(ctx)
	}

	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctxWithTx := context.WithValue(ctx, TxKey, tx)
		return fn(ctxWithTx)
	})
}

// GetDBFromContext extracts the *gorm.DB (transaction or global) from context
// If Txm (TxKey) is present, returns it. Otherwise returns global db passed to repo.
func GetDBFromContext(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(TxKey).(*gorm.DB); ok {
		return tx
	}
	return defaultDB.WithContext(ctx)
}
