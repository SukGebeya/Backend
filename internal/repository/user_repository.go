package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nate/go-boilerplate/internal/db"
	"github.com/nate/go-boilerplate/internal/domain"
	domainerrors "github.com/nate/go-boilerplate/internal/errors"
)

type userRepository struct {
	queries *db.Queries
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{queries: db.New(pool)}
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.UserNotFound("user not found")
		}
		return nil, domainerrors.Internal("failed to get user", err)
	}
	return dbUserToDomain(&user), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := r.queries.GetUserByEmail(ctx, &email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.UserNotFound("user not found")
		}
		return nil, domainerrors.Internal("failed to get user by email", err)
	}
	return dbUserToDomain(&user), nil
}

func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	user, err := r.queries.GetUserByPhone(ctx, &phone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.UserNotFound("user not found")
		}
		return nil, domainerrors.Internal("failed to get user by phone", err)
	}
	return dbUserToDomain(&user), nil
}

func (r *userRepository) GetByEmailOrPhone(ctx context.Context, identifier string) (*domain.User, error) {
	user, err := r.queries.GetUserByEmailOrPhone(ctx, &identifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.UserNotFound("user not found")
		}
		return nil, domainerrors.Internal("failed to get user", err)
	}
	return dbUserToDomain(&user), nil
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	created, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		Phone:        user.Phone,
		PasswordHash: user.PasswordHash,
		Role:         string(user.Role),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create user", err)
	}
	return dbUserToDomain(&created), nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) (*domain.User, error) {
	updated, err := r.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		Phone:        user.Phone,
		PasswordHash: user.PasswordHash,
		Role:         string(user.Role),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.UserNotFound("user not found")
		}
		return nil, domainerrors.Internal("failed to update user", err)
	}
	return dbUserToDomain(&updated), nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteUser(ctx, id); err != nil {
		return domainerrors.Internal("failed to delete user", err)
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.User], error) {
	users, err := r.queries.ListUsers(ctx, db.ListUsersParams{
		Limit:  int32(opts.Limit()),
		Offset: int32(opts.Offset()),
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
		items[i] = *dbUserToDomain(&u)
	}

	return &domain.PaginatedResult[domain.User]{
		Items:      items,
		Total:      count,
		Page:       opts.Page,
		PerPage:    opts.PerPage,
		TotalPages: int((count + int64(opts.PerPage) - 1) / int64(opts.PerPage)),
	}, nil
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		return 0, domainerrors.Internal("failed to count users", err)
	}
	return count, nil
}

func dbUserToDomain(u *db.User) *domain.User {
	return &domain.User{
		ID:           u.ID,
		Email:        u.Email,
		Phone:        u.Phone,
		PasswordHash: u.PasswordHash,
		Role:         domain.Role(u.Role),
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}
