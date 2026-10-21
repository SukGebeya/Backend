package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleBuyer    Role = "buyer"
	RoleMerchant Role = "merchant"
	RoleAdmin    Role = "admin"
)

func (r Role) IsValid() bool {
	return r == RoleBuyer || r == RoleMerchant || r == RoleAdmin
}

type KYCStatus string

const (
	KYCStatusPending   KYCStatus = "pending"
	KYCStatusSubmitted KYCStatus = "submitted"
	KYCStatusVerified  KYCStatus = "verified"
	KYCStatusRejected  KYCStatus = "rejected"
)

func (s KYCStatus) IsValid() bool {
	return s == KYCStatusPending || s == KYCStatusSubmitted || s == KYCStatusVerified || s == KYCStatusRejected
}

type User struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Email        *string   `db:"email" json:"email"`
	Phone        *string   `db:"phone" json:"phone"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         Role      `db:"role" json:"role"`
	FullName     *string   `db:"full_name" json:"full_name"`
	AvatarURL    *string   `db:"avatar_url" json:"avatar_url"`
	KYCStatus    KYCStatus `db:"kyc_status" json:"kyc_status"`
	NationalID   *string   `db:"national_id" json:"national_id"`
	IsActive     bool      `db:"is_active" json:"is_active"`

	PhoneVerifiedAt *time.Time `db:"phone_verified_at" json:"phone_verified_at,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

func (u *User) DisplayName() string {
	if u.FullName != nil && *u.FullName != "" {
		return *u.FullName
	}
	if u.Email != nil {
		return *u.Email
	}
	if u.Phone != nil {
		return *u.Phone
	}
	return u.ID.String()
}

func (u *User) IsVerified() bool {
	return u.KYCStatus == KYCStatusVerified
}
