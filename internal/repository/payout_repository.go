package repository

import (
	"context"
	"errors"
	"time"

	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type payoutRepository struct {
	queries *db.Queries
}

func NewPayoutRepository(pool *pgxpool.Pool) PayoutRepository {
	return &payoutRepository{queries: db.New(pool)}
}

var _ PayoutRepository = (*payoutRepository)(nil)

func (r *payoutRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payout, error) {
	payout, err := r.queries.GetPayoutByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("payout not found")
		}
		return nil, domainerrors.Internal("failed to get payout", err)
	}
	return dbPayoutToDomain(payout), nil
}

func (r *payoutRepository) ListByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]domain.Payout, error) {
	offset := int32((page - 1) * pageSize)

	payouts, err := r.queries.ListPayoutsByMerchantID(ctx, db.ListPayoutsByMerchantIDParams{
		MerchantID: merchantID,
		Offset:     offset,
		Limit:      int32(pageSize),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list payouts", err)
	}

	result := make([]domain.Payout, len(payouts))
	for i, payout := range payouts {
		result[i] = *dbPayoutToDomain(payout)
	}
	return result, nil
}

func (r *payoutRepository) ListByStatus(ctx context.Context, status domain.PayoutStatus, page, pageSize int) ([]domain.Payout, error) {
	offset := int32((page - 1) * pageSize)

	payouts, err := r.queries.ListPayoutsByStatus(ctx, db.ListPayoutsByStatusParams{
		Status: string(status),
		Offset: offset,
		Limit:  int32(pageSize),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list payouts by status", err)
	}

	result := make([]domain.Payout, len(payouts))
	for i, payout := range payouts {
		result[i] = *dbPayoutToDomain(payout)
	}
	return result, nil
}

func (r *payoutRepository) ListAll(ctx context.Context, page, pageSize int) ([]domain.Payout, error) {
	offset := int32((page - 1) * pageSize)

	payouts, err := r.queries.ListAllPayouts(ctx, db.ListAllPayoutsParams{
		Offset: offset,
		Limit:  int32(pageSize),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list all payouts", err)
	}

	result := make([]domain.Payout, len(payouts))
	for i, payout := range payouts {
		result[i] = *dbPayoutToDomain(payout)
	}
	return result, nil
}

func (r *payoutRepository) Create(ctx context.Context, payout *domain.Payout) (*domain.Payout, error) {
	created, err := r.queries.CreatePayout(ctx, db.CreatePayoutParams{
		ID:          payout.ID,
		MerchantID:  payout.MerchantID,
		AmountCents: payout.AmountCents,
		Currency:    payout.Currency,
		Method:      string(payout.Method),
		Destination: payout.Destination,
		Status:      string(payout.Status),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create payout", err)
	}
	return dbPayoutToDomain(created), nil
}

func (r *payoutRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.PayoutStatus, blnkTxID *string, failureReason *string) (*domain.Payout, error) {
	var processedAt pgtype.Timestamptz
	if status == domain.PayoutStatusCompleted || status == domain.PayoutStatusFailed {
		processedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}

	payout, err := r.queries.UpdatePayoutStatus(ctx, db.UpdatePayoutStatusParams{
		ID:                id,
		Status:            string(status),
		BlnkTransactionID: blnkTxID,
		FailureReason:     failureReason,
		ProcessedAt:       processedAt,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("payout not found")
		}
		return nil, domainerrors.Internal("failed to update payout status", err)
	}
	return dbPayoutToDomain(payout), nil
}

func (r *payoutRepository) CountByMerchantID(ctx context.Context, merchantID uuid.UUID) (int64, error) {
	count, err := r.queries.CountPayoutsByMerchantID(ctx, merchantID)
	if err != nil {
		return 0, domainerrors.Internal("failed to count payouts", err)
	}
	return count, nil
}

func (r *payoutRepository) CountByStatus(ctx context.Context, status domain.PayoutStatus) (int64, error) {
	count, err := r.queries.CountPayoutsByStatus(ctx, string(status))
	if err != nil {
		return 0, domainerrors.Internal("failed to count payouts by status", err)
	}
	return count, nil
}

func dbPayoutToDomain(p db.Payout) *domain.Payout {
	var blnkTxID *string
	if p.BlnkTransactionID != nil {
		blnkTxID = p.BlnkTransactionID
	}

	var failureReason *string
	if p.FailureReason != nil {
		failureReason = p.FailureReason
	}

	var processedAt *time.Time
	if p.ProcessedAt.Valid {
		processedAt = &p.ProcessedAt.Time
	}

	return &domain.Payout{
		ID:                p.ID,
		MerchantID:        p.MerchantID,
		AmountCents:       p.AmountCents,
		Currency:          p.Currency,
		Method:            domain.PayoutMethod(p.Method),
		Destination:       p.Destination,
		Status:            domain.PayoutStatus(p.Status),
		BlnkTransactionID: blnkTxID,
		FailureReason:     failureReason,
		RequestedAt:       p.RequestedAt.Time,
		ProcessedAt:       processedAt,
	}
}
