package testutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/nate/go-boilerplate/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type UserBuilder struct {
	user *domain.User
}

func NewUserBuilder() *UserBuilder {
	email := "test@example.com"
	phone := "+1234567890"
	return &UserBuilder{
		user: &domain.User{
			ID:           uuid.New(),
			Email:        &email,
			Phone:        &phone,
			PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3/ItB/XBG/eCknfIrqS6",
			Role:         domain.RoleUser,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}
}

func (b *UserBuilder) WithID(id uuid.UUID) *UserBuilder {
	b.user.ID = id
	return b
}

func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.user.Email = &email
	return b
}

func (b *UserBuilder) WithPhone(phone string) *UserBuilder {
	b.user.Phone = &phone
	return b
}

func (b *UserBuilder) WithPassword(password string) *UserBuilder {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	b.user.PasswordHash = string(hash)
	return b
}

func (b *UserBuilder) WithRole(role domain.Role) *UserBuilder {
	b.user.Role = role
	return b
}

func (b *UserBuilder) WithCreatedAt(createdAt time.Time) *UserBuilder {
	b.user.CreatedAt = createdAt
	return b
}

func (b *UserBuilder) WithUpdatedAt(updatedAt time.Time) *UserBuilder {
	b.user.UpdatedAt = updatedAt
	return b
}

func (b *UserBuilder) Build() *domain.User {
	return b.user
}

type NoteBuilder struct {
	note *domain.Note
}

func NewNoteBuilder() *NoteBuilder {
	return &NoteBuilder{
		note: &domain.Note{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Title:     "Test Note",
			Content:   "This is a test note",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			DeletedAt: nil,
		},
	}
}

func (b *NoteBuilder) WithID(id uuid.UUID) *NoteBuilder {
	b.note.ID = id
	return b
}

func (b *NoteBuilder) WithUserID(userID uuid.UUID) *NoteBuilder {
	b.note.UserID = userID
	return b
}

func (b *NoteBuilder) WithTitle(title string) *NoteBuilder {
	b.note.Title = title
	return b
}

func (b *NoteBuilder) WithContent(content string) *NoteBuilder {
	b.note.Content = content
	return b
}

func (b *NoteBuilder) WithCreatedAt(createdAt time.Time) *NoteBuilder {
	b.note.CreatedAt = createdAt
	return b
}

func (b *NoteBuilder) WithUpdatedAt(updatedAt time.Time) *NoteBuilder {
	b.note.UpdatedAt = updatedAt
	return b
}

func (b *NoteBuilder) WithDeletedAt(deletedAt time.Time) *NoteBuilder {
	b.note.DeletedAt = &deletedAt
	return b
}

func (b *NoteBuilder) Deleted() *NoteBuilder {
	now := time.Now()
	b.note.DeletedAt = &now
	return b
}

func (b *NoteBuilder) Build() *domain.Note {
	return b.note
}
