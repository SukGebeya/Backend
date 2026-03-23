package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

func (r Role) IsValid() bool {
	return r == RoleUser || r == RoleAdmin
}

type User struct {
	ID           uuid.UUID
	Email        *string
	Phone        *string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) DisplayName() string {
	if u.Email != nil {
		return *u.Email
	}
	if u.Phone != nil {
		return *u.Phone
	}
	return u.ID.String()
}
