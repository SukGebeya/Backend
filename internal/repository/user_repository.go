package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
)

type userRepository struct {
	queries *db.Queries
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{queries: db.New(pool)}
}

func uuidToPgType(id uuid.UUID) pgtype.UUID {
	var pgID pgtype.UUID
	_ = pgID.Scan(id.String())
	return pgID
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := r.queries.GetUserByID(ctx, uuidToPgType(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.UserNotFound("user not found")
		}
		return nil, domainerrors.Internal("failed to get user", err)
	}
	return dbUserRowToDomain(&user), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := r.queries.GetUserByEmail(ctx, &email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.UserNotFound("user not found")
		}
		return nil, domainerrors.Internal("failed to get user by email", err)
	}
	return dbGetUserByEmailRowToDomain(&user), nil
}

func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	user, err := r.queries.GetUserByPhone(ctx, &phone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.UserNotFound("user not found")
		}
		return nil, domainerrors.Internal("failed to get user by phone", err)
	}
	return dbGetUserByPhoneRowToDomain(&user), nil
}

func (r *userRepository) GetByEmailOrPhone(ctx context.Context, identifier string) (*domain.User, error) {
	user, err := r.queries.GetUserByEmailOrPhone(ctx, &identifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.UserNotFound("user not found")
		}
		return nil, domainerrors.Internal("failed to get user", err)
	}
	return dbGetUserByEmailOrPhoneRowToDomain(&user), nil
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	role := string(user.Role)
	kycStatus := string(user.KYCStatus)
	passwordHash := user.PasswordHash
	isActive := user.IsActive

	created, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		ID:           uuidToPgType(user.ID),
		Email:        user.Email,
		Phone:        user.Phone,
		PasswordHash: &passwordHash,
		Role:         &role,
		KycStatus:    &kycStatus,
		IsActive:     &isActive,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create user", err)
	}
	return dbCreateUserRowToDomain(&created), nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) (*domain.User, error) {
	role := string(user.Role)
	kycStatus := string(user.KYCStatus)
	passwordHash := user.PasswordHash
	isActive := user.IsActive

	updated, err := r.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:           uuidToPgType(user.ID),
		Email:        user.Email,
		Phone:        user.Phone,
		PasswordHash: &passwordHash,
		Role:         &role,
		FullName:     user.FullName,
		AvatarUrl:    user.AvatarURL,
		KycStatus:    &kycStatus,
		NationalID:   user.NationalID,
		IsActive:     &isActive,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.UserNotFound("user not found")
		}
		return nil, domainerrors.Internal("failed to update user", err)
	}
	return dbUpdateUserRowToDomain(&updated), nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteUser(ctx, uuidToPgType(id)); err != nil {
		return domainerrors.Internal("failed to delete user", err)
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.User], error) {
	limit := int64(opts.Limit())
	offset := int64(opts.Offset())

	users, err := r.queries.ListUsers(ctx, db.ListUsersParams{
		PageLimit:  &limit,
		PageOffset: &offset,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list users", err)
	}

	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		return nil, domainerrors.Internal("failed to count users", err)
	}

	items := make([]domain.User, len(users))
	for i, u := range users {
		items[i] = *dbListUsersRowToDomain(&u)
	}

	var totalCount int64
	if count != nil {
		totalCount = *count
	}

	return &domain.PaginatedResult[domain.User]{
		Items:      items,
		Total:      totalCount,
		Page:       opts.Page,
		PerPage:    opts.PerPage,
		TotalPages: int((totalCount + int64(opts.PerPage) - 1) / int64(opts.PerPage)),
	}, nil
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		return 0, domainerrors.Internal("failed to count users", err)
	}
	if count == nil {
		return 0, nil
	}
	return *count, nil
}

func dbUserRowToDomain(u *db.GetUserByIDRow) *domain.User {
	return &domain.User{
		ID:           u.ID,
		Email:        u.Email,
		Phone:        u.Phone,
		PasswordHash: u.PasswordHash,
		Role:         domain.Role(u.Role),
		FullName:     u.FullName,
		AvatarURL:    u.AvatarUrl,
		KYCStatus:    domain.KYCStatus(u.KycStatus),
		NationalID:   u.NationalID,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}

func dbGetUserByEmailRowToDomain(u *db.GetUserByEmailRow) *domain.User {
	return &domain.User{
		ID:           u.ID,
		Email:        u.Email,
		Phone:        u.Phone,
		PasswordHash: u.PasswordHash,
		Role:         domain.Role(u.Role),
		FullName:     u.FullName,
		AvatarURL:    u.AvatarUrl,
		KYCStatus:    domain.KYCStatus(u.KycStatus),
		NationalID:   u.NationalID,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}

func dbGetUserByPhoneRowToDomain(u *db.GetUserByPhoneRow) *domain.User {
	return &domain.User{
		ID:           u.ID,
		Email:        u.Email,
		Phone:        u.Phone,
		PasswordHash: u.PasswordHash,
		Role:         domain.Role(u.Role),
		FullName:     u.FullName,
		AvatarURL:    u.AvatarUrl,
		KYCStatus:    domain.KYCStatus(u.KycStatus),
		NationalID:   u.NationalID,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}

func dbGetUserByEmailOrPhoneRowToDomain(u *db.GetUserByEmailOrPhoneRow) *domain.User {
	return &domain.User{
		ID:           u.ID,
		Email:        u.Email,
		Phone:        u.Phone,
		PasswordHash: u.PasswordHash,
		Role:         domain.Role(u.Role),
		FullName:     u.FullName,
		AvatarURL:    u.AvatarUrl,
		KYCStatus:    domain.KYCStatus(u.KycStatus),
		NationalID:   u.NationalID,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}

func dbCreateUserRowToDomain(u *db.CreateUserRow) *domain.User {
	return &domain.User{
		ID:           u.ID,
		Email:        u.Email,
		Phone:        u.Phone,
		PasswordHash: u.PasswordHash,
		Role:         domain.Role(u.Role),
		FullName:     u.FullName,
		AvatarURL:    u.AvatarUrl,
		KYCStatus:    domain.KYCStatus(u.KycStatus),
		NationalID:   u.NationalID,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}

func dbUpdateUserRowToDomain(u *db.UpdateUserRow) *domain.User {
	return &domain.User{
		ID:           u.ID,
		Email:        u.Email,
		Phone:        u.Phone,
		PasswordHash: u.PasswordHash,
		Role:         domain.Role(u.Role),
		FullName:     u.FullName,
		AvatarURL:    u.AvatarUrl,
		KYCStatus:    domain.KYCStatus(u.KycStatus),
		NationalID:   u.NationalID,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}

func dbListUsersRowToDomain(u *db.ListUsersRow) *domain.User {
	var createdAt, updatedAt time.Time
	if u.CreatedAt.Valid {
		createdAt = u.CreatedAt.Time
	}
	if u.UpdatedAt.Valid {
		updatedAt = u.UpdatedAt.Time
	}
	return &domain.User{
		ID:           u.ID,
		Email:        u.Email,
		Phone:        u.Phone,
		PasswordHash: u.PasswordHash,
		Role:         domain.Role(u.Role),
		FullName:     u.FullName,
		AvatarURL:    u.AvatarUrl,
		KYCStatus:    domain.KYCStatus(u.KycStatus),
		NationalID:   u.NationalID,
		IsActive:     u.IsActive,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}
