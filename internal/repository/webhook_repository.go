package repository

import (
	"context"
	"errors"

	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type webhookEventRepository struct {
	queries *db.Queries
}

var _ WebhookEventRepository = (*webhookEventRepository)(nil)

func NewWebhookEventRepository(pool *pgxpool.Pool) WebhookEventRepository {
	return &webhookEventRepository{queries: db.New(pool)}
}

func (r *webhookEventRepository) WithTx(tx pgx.Tx) WebhookEventRepository {
	return &webhookEventRepository{queries: r.queries.WithTx(tx)}
}

// Create returns created=false when an identical payload was already recorded.
func (r *webhookEventRepository) Create(ctx context.Context, event *domain.WebhookEvent) (*domain.WebhookEvent, bool, error) {
	row, err := r.queries.CreateWebhookEvent(ctx, db.CreateWebhookEventParams{
		ID:          event.ID,
		Provider:    event.Provider,
		EventType:   event.EventType,
		TxRef:       event.TxRef,
		PayloadHash: event.PayloadHash,
		Payload:     event.Payload,
		Status:      string(event.Status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, domainerrors.Internal("failed to record webhook event", err)
	}
	return dbWebhookEventToDomain(row), true, nil
}

func (r *webhookEventRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.WebhookEvent, error) {
	row, err := r.queries.GetWebhookEventByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("webhook event not found")
		}
		return nil, domainerrors.Internal("failed to get webhook event", err)
	}
	return dbWebhookEventToDomain(row), nil
}

func (r *webhookEventRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.WebhookEventStatus, errMsg *string) error {
	if err := r.queries.UpdateWebhookEventStatus(ctx, db.UpdateWebhookEventStatusParams{
		ID:     id,
		Status: string(status),
		Error:  errMsg,
	}); err != nil {
		return domainerrors.Internal("failed to update webhook event", err)
	}
	return nil
}

func dbWebhookEventToDomain(e db.WebhookEvent) *domain.WebhookEvent {
	event := &domain.WebhookEvent{
		ID:          e.ID,
		Provider:    e.Provider,
		EventType:   e.EventType,
		TxRef:       e.TxRef,
		PayloadHash: e.PayloadHash,
		Payload:     e.Payload,
		Status:      domain.WebhookEventStatus(e.Status),
		Error:       e.Error,
		ReceivedAt:  e.ReceivedAt.Time,
	}
	if e.ProcessedAt.Valid {
		event.ProcessedAt = &e.ProcessedAt.Time
	}
	return event
}
