package service

import (
	"context"

	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
)

var _ CategoryManager = (*CategoryService)(nil)

type CategoryService struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryService(categoryRepo repository.CategoryRepository) *CategoryService {
	return &CategoryService{categoryRepo: categoryRepo}
}

func (s *CategoryService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	return s.categoryRepo.GetByID(ctx, id)
}

func (s *CategoryService) GetBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	return s.categoryRepo.GetBySlug(ctx, slug)
}

func (s *CategoryService) List(ctx context.Context) ([]domain.Category, error) {
	return s.categoryRepo.List(ctx)
}

func (s *CategoryService) ListByParent(ctx context.Context, parentID uuid.UUID) ([]domain.Category, error) {
	return s.categoryRepo.ListByParent(ctx, parentID)
}

func (s *CategoryService) ListTopLevel(ctx context.Context) ([]domain.Category, error) {
	return s.categoryRepo.ListTopLevel(ctx)
}

func (s *CategoryService) Create(ctx context.Context, input CreateCategoryInput) (*domain.Category, error) {
	existing, _ := s.categoryRepo.GetBySlug(ctx, input.Slug)
	if existing != nil {
		return nil, domainerrors.Conflict("category slug already exists")
	}

	var parentID *uuid.UUID
	if input.ParentID != nil && *input.ParentID != "" {
		pid, err := uuid.Parse(*input.ParentID)
		if err != nil {
			return nil, domainerrors.BadRequest("invalid parent category ID")
		}
		parentID = &pid
	}

	category := &domain.Category{
		ID:        uuid.New(),
		ParentID:  parentID,
		Name:      input.Name,
		Slug:      input.Slug,
		IconURL:   input.IconURL,
		IsActive:  input.IsActive,
		SortOrder: input.SortOrder,
	}

	return s.categoryRepo.Create(ctx, category)
}

func (s *CategoryService) Update(ctx context.Context, id uuid.UUID, input UpdateCategoryInput) (*domain.Category, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		category.Name = *input.Name
	}
	if input.Slug != nil {
		existing, _ := s.categoryRepo.GetBySlug(ctx, *input.Slug)
		if existing != nil && existing.ID != id {
			return nil, domainerrors.Conflict("category slug already exists")
		}
		category.Slug = *input.Slug
	}
	if input.IconURL != nil {
		category.IconURL = input.IconURL
	}
	if input.IsActive != nil {
		category.IsActive = *input.IsActive
	}
	if input.SortOrder != nil {
		category.SortOrder = *input.SortOrder
	}

	return s.categoryRepo.Update(ctx, category)
}

func (s *CategoryService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.categoryRepo.Delete(ctx, id)
}
