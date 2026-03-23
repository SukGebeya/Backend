package repository

import (
	"context"
	"errors"

	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type categoryRepository struct {
	queries *db.Queries
}

func NewCategoryRepository(pool *pgxpool.Pool) CategoryRepository {
	return &categoryRepository{queries: db.New(pool)}
}

var _ CategoryRepository = (*categoryRepository)(nil)

func (r *categoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	category, err := r.queries.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("category not found")
		}
		return nil, domainerrors.Internal("failed to get category", err)
	}
	return dbCategoryToDomain(category), nil
}

func (r *categoryRepository) GetBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	category, err := r.queries.GetCategoryBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("category not found")
		}
		return nil, domainerrors.Internal("failed to get category by slug", err)
	}
	return dbCategoryToDomain(category), nil
}

func (r *categoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	categories, err := r.queries.ListCategories(ctx)
	if err != nil {
		return nil, domainerrors.Internal("failed to list categories", err)
	}

	result := make([]domain.Category, len(categories))
	for i, c := range categories {
		result[i] = *dbCategoryToDomain(c)
	}
	return result, nil
}

func (r *categoryRepository) ListByParent(ctx context.Context, parentID uuid.UUID) ([]domain.Category, error) {
	var pgParentID pgtype.UUID
	_ = pgParentID.Scan(parentID.String())

	categories, err := r.queries.ListCategoriesByParent(ctx, pgParentID)
	if err != nil {
		return nil, domainerrors.Internal("failed to list categories by parent", err)
	}

	result := make([]domain.Category, len(categories))
	for i, c := range categories {
		result[i] = *dbCategoryToDomain(c)
	}
	return result, nil
}

func (r *categoryRepository) ListTopLevel(ctx context.Context) ([]domain.Category, error) {
	categories, err := r.queries.ListTopLevelCategories(ctx)
	if err != nil {
		return nil, domainerrors.Internal("failed to list top level categories", err)
	}

	result := make([]domain.Category, len(categories))
	for i, c := range categories {
		result[i] = *dbCategoryToDomain(c)
	}
	return result, nil
}

func (r *categoryRepository) Create(ctx context.Context, category *domain.Category) (*domain.Category, error) {
	var pgParentID pgtype.UUID
	if category.ParentID != nil {
		_ = pgParentID.Scan(category.ParentID.String())
	}

	created, err := r.queries.CreateCategory(ctx, db.CreateCategoryParams{
		ID:        category.ID,
		ParentID:  pgParentID,
		Name:      category.Name,
		Slug:      category.Slug,
		IconUrl:   category.IconURL,
		IsActive:  category.IsActive,
		SortOrder: int32(category.SortOrder),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create category", err)
	}
	return dbCategoryToDomain(created), nil
}

func (r *categoryRepository) Update(ctx context.Context, category *domain.Category) (*domain.Category, error) {
	updated, err := r.queries.UpdateCategory(ctx, db.UpdateCategoryParams{
		ID:        category.ID,
		Name:      category.Name,
		Slug:      category.Slug,
		IconUrl:   category.IconURL,
		IsActive:  category.IsActive,
		SortOrder: int32(category.SortOrder),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("category not found")
		}
		return nil, domainerrors.Internal("failed to update category", err)
	}
	return dbCategoryToDomain(updated), nil
}

func (r *categoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteCategory(ctx, id)
}

func dbCategoryToDomain(c db.Category) *domain.Category {
	var parentID *uuid.UUID
	if c.ParentID.Valid {
		pid, _ := uuid.FromBytes(c.ParentID.Bytes[:])
		parentID = &pid
	}

	return &domain.Category{
		ID:        c.ID,
		ParentID:  parentID,
		Name:      c.Name,
		Slug:      c.Slug,
		IconURL:   c.IconUrl,
		IsActive:  c.IsActive,
		SortOrder: int(c.SortOrder),
		CreatedAt: c.CreatedAt.Time,
	}
}
