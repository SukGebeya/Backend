package repository

import (
	"context"
	"errors"

	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type walletTransactionRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewWalletTransactionRepository(pool *pgxpool.Pool) WalletTransactionRepository {
	return &walletTransactionRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

var _ WalletTransactionRepository = (*walletTransactionRepository)(nil)

func (r *walletTransactionRepository) WithTx(tx pgx.Tx) WalletTransactionRepository {
	return &walletTransactionRepository{
		pool:    r.pool,
		queries: r.queries.WithTx(tx),
	}
}

func (r *walletTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.WalletTransaction, error) {
	txn, err := r.queries.GetWalletTransactionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("wallet transaction not found")
		}
		return nil, domainerrors.Internal("failed to get wallet transaction", err)
	}
	return dbWalletTransactionToDomain(txn), nil
}

func (r *walletTransactionRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.WalletTransaction, error) {
	var pgOrderID pgtype.UUID
	pgOrderID = pgtype.UUID{Bytes: orderID, Valid: true}

	txn, err := r.queries.GetWalletTransactionByOrderID(ctx, pgOrderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("wallet transaction not found")
		}
		return nil, domainerrors.Internal("failed to get wallet transaction", err)
	}
	return dbWalletTransactionToDomain(txn), nil
}

func (r *walletTransactionRepository) ListByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]domain.WalletTransaction, error) {
	offset := int32((page - 1) * pageSize)

	txns, err := r.queries.ListWalletTransactionsByMerchantID(ctx, db.ListWalletTransactionsByMerchantIDParams{
		MerchantID: merchantID,
		PageOffset: offset,
		PageSize:   int32(pageSize),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list wallet transactions", err)
	}

	result := make([]domain.WalletTransaction, len(txns))
	for i, txn := range txns {
		result[i] = *dbWalletTransactionToDomain(txn)
	}
	return result, nil
}

func (r *walletTransactionRepository) Create(ctx context.Context, txn *domain.WalletTransaction) (*domain.WalletTransaction, error) {
	var orderID pgtype.UUID
	if txn.OrderID != nil {
		orderID = pgtype.UUID{Bytes: *txn.OrderID, Valid: true}
	}

	var description *string
	if txn.Description != nil {
		description = txn.Description
	}

	created, err := r.queries.CreateWalletTransaction(ctx, db.CreateWalletTransactionParams{
		ID:                txn.ID,
		MerchantID:        txn.MerchantID,
		OrderID:           orderID,
		Type:              string(txn.Type),
		AmountCents:       txn.AmountCents,
		Currency:          txn.Currency,
		BlnkTransactionID: txn.BlnkTransactionID,
		Status:            string(txn.Status),
		Description:       description,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create wallet transaction", err)
	}
	return dbWalletTransactionToDomain(created), nil
}

func (r *walletTransactionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.WalletTransactionStatus) (*domain.WalletTransaction, error) {
	txn, err := r.queries.UpdateWalletTransactionStatus(ctx, db.UpdateWalletTransactionStatusParams{
		Status: string(status),
		ID:     id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("wallet transaction not found")
		}
		return nil, domainerrors.Internal("failed to update wallet transaction status", err)
	}
	return dbWalletTransactionToDomain(txn), nil
}

func (r *walletTransactionRepository) CountByMerchantID(ctx context.Context, merchantID uuid.UUID) (int64, error) {
	count, err := r.queries.CountWalletTransactionsByMerchantID(ctx, merchantID)
	if err != nil {
		return 0, domainerrors.Internal("failed to count wallet transactions", err)
	}
	return count, nil
}

func dbWalletTransactionToDomain(t db.WalletTransaction) *domain.WalletTransaction {
	var orderID *uuid.UUID
	if t.OrderID.Valid {
		oid := uuid.UUID(t.OrderID.Bytes)
		orderID = &oid
	}

	var description *string
	if t.Description != nil {
		description = t.Description
	}

	return &domain.WalletTransaction{
		ID:                t.ID,
		MerchantID:        t.MerchantID,
		OrderID:           orderID,
		Type:              domain.WalletTransactionType(t.Type),
		AmountCents:       t.AmountCents,
		Currency:          t.Currency,
		BlnkTransactionID: t.BlnkTransactionID,
		Status:            domain.WalletTransactionStatus(t.Status),
		Description:       description,
		CreatedAt:         t.CreatedAt.Time,
	}
}
