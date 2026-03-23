package testutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/SukGebeya/Backend/internal/domain"
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
			Role:         domain.RoleBuyer,
			KYCStatus:    domain.KYCStatusPending,
			IsActive:     true,
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
