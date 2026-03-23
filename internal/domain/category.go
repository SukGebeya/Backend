package domain

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	ParentID  *uuid.UUID `db:"parent_id" json:"parent_id,omitempty"`
	Name      string     `db:"name" json:"name"`
	Slug      string     `db:"slug" json:"slug"`
	IconURL   *string    `db:"icon_url" json:"icon_url,omitempty"`
	IsActive  bool       `db:"is_active" json:"is_active"`
	SortOrder int        `db:"sort_order" json:"sort_order"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

func (c *Category) IsTopLevel() bool {
	return c.ParentID == nil
}
