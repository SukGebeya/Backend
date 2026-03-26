package domain

import (
	"time"

	"github.com/google/uuid"
)

type Address struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	FullName  string    `db:"full_name" json:"full_name"`
	Phone     string    `db:"phone" json:"phone"`
	Line1     string    `db:"line1" json:"line1"`
	Line2     *string   `db:"line2" json:"line2,omitempty"`
	City      string    `db:"city" json:"city"`
	Subcity   *string   `db:"subcity" json:"subcity,omitempty"`
	Woreda    *string   `db:"woreda" json:"woreda,omitempty"`
	IsDefault bool      `db:"is_default" json:"is_default"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type CreateAddressInput struct {
	UserID    uuid.UUID
	FullName  string
	Phone     string
	Line1     string
	Line2     *string
	City      string
	Subcity   *string
	Woreda    *string
	IsDefault bool
}

type UpdateAddressInput struct {
	FullName  *string
	Phone     *string
	Line1     *string
	Line2     *string
	City      *string
	Subcity   *string
	Woreda    *string
	IsDefault *bool
}
