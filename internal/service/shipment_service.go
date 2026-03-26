package service

import (
	"context"
	"time"

	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
)

type ShipmentManager interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Shipment, error)
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Shipment, error)
	CreateForOrder(ctx context.Context, orderID uuid.UUID) (*domain.Shipment, error)
	MarkShipped(ctx context.Context, id uuid.UUID, carrier, trackingNumber *string, estimatedDelivery *time.Time) (*domain.Shipment, error)
	MarkDelivered(ctx context.Context, id uuid.UUID) (*domain.Shipment, error)
}

type ShipmentService struct {
	shipmentRepo repository.ShipmentRepository
}

func NewShipmentService(shipmentRepo repository.ShipmentRepository) *ShipmentService {
	return &ShipmentService{
		shipmentRepo: shipmentRepo,
	}
}

var _ ShipmentManager = (*ShipmentService)(nil)

func (s *ShipmentService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Shipment, error) {
	return s.shipmentRepo.GetByID(ctx, id)
}

func (s *ShipmentService) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Shipment, error) {
	return s.shipmentRepo.GetByOrderID(ctx, orderID)
}

func (s *ShipmentService) CreateForOrder(ctx context.Context, orderID uuid.UUID) (*domain.Shipment, error) {
	shipment := &domain.Shipment{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  domain.ShipmentStatusPreparing,
	}

	created, err := s.shipmentRepo.Create(ctx, shipment)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *ShipmentService) MarkShipped(ctx context.Context, id uuid.UUID, carrier, trackingNumber *string, estimatedDelivery *time.Time) (*domain.Shipment, error) {
	existing, err := s.shipmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.Status != domain.ShipmentStatusPreparing && existing.Status != domain.ShipmentStatusShipped {
		return nil, domainerrors.BadRequest("shipment must be in preparing status to mark as shipped")
	}

	return s.shipmentRepo.UpdateShipped(ctx, id, carrier, trackingNumber, estimatedDelivery)
}

func (s *ShipmentService) MarkDelivered(ctx context.Context, id uuid.UUID) (*domain.Shipment, error) {
	existing, err := s.shipmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.Status != domain.ShipmentStatusShipped {
		return nil, domainerrors.BadRequest("shipment must be in shipped status to mark as delivered")
	}

	return s.shipmentRepo.UpdateDelivered(ctx, id)
}
