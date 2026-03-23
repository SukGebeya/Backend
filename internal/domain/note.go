package domain

import (
	"time"

	"github.com/google/uuid"
)

type Note struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Title     string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (n *Note) IsDeleted() bool {
	return n.DeletedAt != nil
}

type NoteListOptions struct {
	ListOptions
	WithDeleted bool
}
