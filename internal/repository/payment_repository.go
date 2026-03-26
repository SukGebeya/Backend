package repository

import (
	"context"
	"encoding/json"

	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type paymentRepository struct {
	queries *db.Queries
}

func NewPaymentRepository(pool *pgxpool.Pool) PaymentRepository {
	return &paymentRepository{queries: db.New(pool)}
}

var _ PaymentRepository = (*paymentRepository)(nil)

func (r *paymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	payment, err := r.queries.GetPaymentByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("payment not found")
		}
		return nil, domainerrors.Internal("failed to get payment", err)
	}
	return dbPaymentToDomain(payment), nil
}

func (r *paymentRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Payment, error) {
	payment, err := r.queries.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("payment not found")
		}
		return nil, domainerrors.Internal("failed to get payment", err)
	}
	return dbPaymentToDomain(payment), nil
}

func (r *paymentRepository) GetByProviderTxRef(ctx context.Context, providerTxRef string) (*domain.Payment, error) {
	payment, err := r.queries.GetPaymentByProviderTxRef(ctx, &providerTxRef)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("payment not found")
		}
		return nil, domainerrors.Internal("failed to get payment", err)
	}
	return dbPaymentToDomain(payment), nil
}

func (r *paymentRepository) ListByStatus(ctx context.Context, status domain.PaymentStatus, page, pageSize int) ([]domain.Payment, error) {
	offset := (page - 1) * pageSize
	payments, err := r.queries.ListPaymentsByStatus(ctx, db.ListPaymentsByStatusParams{
		Status:     string(status),
		PageSize:   int32(pageSize),
		PageOffset: int32(offset),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list payments", err)
	}
	result := make([]domain.Payment, len(payments))
	for i, p := range payments {
		result[i] = *dbPaymentToDomain(p)
	}
	return result, nil
}

func (r *paymentRepository) Create(ctx context.Context, payment *domain.Payment) (*domain.Payment, error) {
	var providerTxRef *string
	if payment.ProviderTxRef != nil {
		providerTxRef = payment.ProviderTxRef
	}

	var providerResponse []byte
	if payment.ProviderResponse != nil {
		var err error
		providerResponse, err = json.Marshal(payment.ProviderResponse)
		if err != nil {
			return nil, domainerrors.Internal("failed to marshal provider response", err)
		}
	}

	var blnkTxID *string
	if payment.BlnkTransactionID != nil {
		blnkTxID = payment.BlnkTransactionID
	}

	created, err := r.queries.CreatePayment(ctx, db.CreatePaymentParams{
		ID:                payment.ID,
		OrderID:           payment.OrderID,
		Provider:          payment.Provider,
		Status:            string(payment.Status),
		AmountCents:       payment.AmountCents,
		Currency:          payment.Currency,
		ProviderTxRef:     providerTxRef,
		ProviderResponse:  providerResponse,
		BlnkTransactionID: blnkTxID,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create payment", err)
	}
	return dbPaymentToDomain(created), nil
}

func (r *paymentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.PaymentStatus) (*domain.Payment, error) {
	payment, err := r.queries.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
		ID:     id,
		Status: string(status),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("payment not found")
		}
		return nil, domainerrors.Internal("failed to update payment status", err)
	}
	return dbPaymentToDomain(payment), nil
}

func (r *paymentRepository) UpdateProviderResponse(ctx context.Context, id uuid.UUID, providerResponse map[string]any) (*domain.Payment, error) {
	responseJSON, err := json.Marshal(providerResponse)
	if err != nil {
		return nil, domainerrors.Internal("failed to marshal provider response", err)
	}

	payment, err := r.queries.UpdatePaymentProviderResponse(ctx, db.UpdatePaymentProviderResponseParams{
		ID:               id,
		ProviderResponse: responseJSON,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("payment not found")
		}
		return nil, domainerrors.Internal("failed to update payment", err)
	}
	return dbPaymentToDomain(payment), nil
}

func (r *paymentRepository) UpdateBlnkTransaction(ctx context.Context, id uuid.UUID, blnkTxID string) (*domain.Payment, error) {
	payment, err := r.queries.UpdatePaymentBlnkTransaction(ctx, db.UpdatePaymentBlnkTransactionParams{
		ID:                id,
		BlnkTransactionID: &blnkTxID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("payment not found")
		}
		return nil, domainerrors.Internal("failed to update payment", err)
	}
	return dbPaymentToDomain(payment), nil
}

func dbPaymentToDomain(p db.Payment) *domain.Payment {
	var providerResponse map[string]any
	if p.ProviderResponse != nil {
		_ = json.Unmarshal(p.ProviderResponse, &providerResponse)
	}

	return &domain.Payment{
		ID:                p.ID,
		OrderID:           p.OrderID,
		Provider:          p.Provider,
		Status:            domain.PaymentStatus(p.Status),
		AmountCents:       p.AmountCents,
		Currency:          p.Currency,
		ProviderTxRef:     p.ProviderTxRef,
		ProviderResponse:  providerResponse,
		BlnkTransactionID: p.BlnkTransactionID,
		CreatedAt:         p.CreatedAt.Time,
		UpdatedAt:         p.UpdatedAt.Time,
	}
}
