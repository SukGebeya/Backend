package domain

import (
	"time"

	"github.com/google/uuid"
)

type ShipmentStatus string

const (
	ShipmentStatusPreparing      ShipmentStatus = "preparing"
	ShipmentStatusShipped        ShipmentStatus = "shipped"
	ShipmentStatusOutForDelivery ShipmentStatus = "out_for_delivery"
	ShipmentStatusDelivered      ShipmentStatus = "delivered"
	ShipmentStatusFailed         ShipmentStatus = "failed"
)

func (s ShipmentStatus) IsValid() bool {
	switch s {
	case ShipmentStatusPreparing, ShipmentStatusShipped, ShipmentStatusOutForDelivery,
		ShipmentStatusDelivered, ShipmentStatusFailed:
		return true
	default:
		return false
	}
}

func (s ShipmentStatus) String() string {
	return string(s)
}

type Shipment struct {
	ID                uuid.UUID      `db:"id" json:"id"`
	OrderID           uuid.UUID      `db:"order_id" json:"order_id"`
	Status            ShipmentStatus `db:"status" json:"status"`
	Carrier           *string        `db:"carrier" json:"carrier,omitempty"`
	TrackingNumber    *string        `db:"tracking_number" json:"tracking_number,omitempty"`
	EstimatedDelivery *time.Time     `db:"estimated_delivery" json:"estimated_delivery,omitempty"`
	ShippedAt         *time.Time     `db:"shipped_at" json:"shipped_at,omitempty"`
	DeliveredAt       *time.Time     `db:"delivered_at" json:"delivered_at,omitempty"`
	CreatedAt         time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at" json:"updated_at"`
}

type CreateShipmentInput struct {
	OrderID           uuid.UUID
	Carrier           *string
	TrackingNumber    *string
	EstimatedDelivery *time.Time
}

type UpdateShipmentInput struct {
	Status            *ShipmentStatus
	Carrier           *string
	TrackingNumber    *string
	EstimatedDelivery *time.Time
}

type MarkShippedInput struct {
	Carrier           *string
	TrackingNumber    *string
	EstimatedDelivery *time.Time
}
