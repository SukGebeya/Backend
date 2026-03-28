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

type shipmentRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewShipmentRepository(pool *pgxpool.Pool) ShipmentRepository {
	return &shipmentRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

var _ ShipmentRepository = (*shipmentRepository)(nil)

func (r *shipmentRepository) WithTx(tx pgx.Tx) ShipmentRepository {
	return &shipmentRepository{
		pool:    r.pool,
		queries: r.queries.WithTx(tx),
	}
}

func (r *shipmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Shipment, error) {
	shipment, err := r.queries.GetShipmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("shipment not found")
		}
		return nil, domainerrors.Internal("failed to get shipment", err)
	}
	return dbShipmentToDomain(shipment), nil
}

func (r *shipmentRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Shipment, error) {
	shipment, err := r.queries.GetShipmentByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("shipment not found")
		}
		return nil, domainerrors.Internal("failed to get shipment", err)
	}
	return dbShipmentToDomain(shipment), nil
}

func (r *shipmentRepository) Create(ctx context.Context, shipment *domain.Shipment) (*domain.Shipment, error) {
	var carrier, trackingNumber *string
	if shipment.Carrier != nil {
		carrier = shipment.Carrier
	}
	if shipment.TrackingNumber != nil {
		trackingNumber = shipment.TrackingNumber
	}

	var estimatedDelivery pgtype.Date
	if shipment.EstimatedDelivery != nil {
		estimatedDelivery = pgtype.Date{Time: *shipment.EstimatedDelivery, Valid: true}
	}

	created, err := r.queries.CreateShipment(ctx, db.CreateShipmentParams{
		ID:                shipment.ID,
		OrderID:           shipment.OrderID,
		Status:            db.ShipmentStatus(shipment.Status),
		Carrier:           carrier,
		TrackingNumber:    trackingNumber,
		EstimatedDelivery: estimatedDelivery,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create shipment", err)
	}
	return dbShipmentToDomain(created), nil
}

func (r *shipmentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ShipmentStatus) (*domain.Shipment, error) {
	shipment, err := r.queries.UpdateShipmentStatus(ctx, db.UpdateShipmentStatusParams{
		Status: db.ShipmentStatus(status),
		ID:     id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("shipment not found")
		}
		return nil, domainerrors.Internal("failed to update shipment status", err)
	}
	return dbShipmentToDomain(shipment), nil
}

func (r *shipmentRepository) UpdateShipped(ctx context.Context, id uuid.UUID, carrier, trackingNumber *string, estimatedDelivery *time.Time) (*domain.Shipment, error) {
	var estDelivery pgtype.Date
	if estimatedDelivery != nil {
		estDelivery = pgtype.Date{Time: *estimatedDelivery, Valid: true}
	}

	shipment, err := r.queries.UpdateShipmentShipped(ctx, db.UpdateShipmentShippedParams{
		Carrier:           carrier,
		TrackingNumber:    trackingNumber,
		EstimatedDelivery: estDelivery,
		ID:                id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("shipment not found")
		}
		return nil, domainerrors.Internal("failed to update shipment", err)
	}
	return dbShipmentToDomain(shipment), nil
}

func (r *shipmentRepository) UpdateDelivered(ctx context.Context, id uuid.UUID) (*domain.Shipment, error) {
	shipment, err := r.queries.UpdateShipmentDelivered(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("shipment not found")
		}
		return nil, domainerrors.Internal("failed to update shipment", err)
	}
	return dbShipmentToDomain(shipment), nil
}

func dbShipmentToDomain(s db.Shipment) *domain.Shipment {
	var carrier, trackingNumber *string
	if s.Carrier != nil {
		carrier = s.Carrier
	}
	if s.TrackingNumber != nil {
		trackingNumber = s.TrackingNumber
	}

	var estimatedDelivery *time.Time
	if s.EstimatedDelivery.Valid {
		est := s.EstimatedDelivery.Time
		estimatedDelivery = &est
	}

	var shippedAt, deliveredAt *time.Time
	if s.ShippedAt.Valid {
		sa := s.ShippedAt.Time
		shippedAt = &sa
	}
	if s.DeliveredAt.Valid {
		da := s.DeliveredAt.Time
		deliveredAt = &da
	}

	return &domain.Shipment{
		ID:                s.ID,
		OrderID:           s.OrderID,
		Status:            domain.ShipmentStatus(s.Status),
		Carrier:           carrier,
		TrackingNumber:    trackingNumber,
		EstimatedDelivery: estimatedDelivery,
		ShippedAt:         shippedAt,
		DeliveredAt:       deliveredAt,
		CreatedAt:         s.CreatedAt.Time,
		UpdatedAt:         s.UpdatedAt.Time,
	}
}
