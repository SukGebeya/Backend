package domain

import (
	"time"

	"github.com/google/uuid"
)

type Inventory struct {
	ID        uuid.UUID `db:"id" json:"id"`
	VariantID uuid.UUID `db:"variant_id" json:"variant_id"`
	Quantity  int       `db:"quantity" json:"quantity"`
	Reserved  int       `db:"reserved" json:"reserved"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (i *Inventory) Available() int {
	return i.Quantity - i.Reserved
}

func (i *Inventory) CanReserve(qty int) bool {
	return i.Available() >= qty
}
