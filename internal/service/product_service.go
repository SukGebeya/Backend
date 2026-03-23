package service

import (
	"context"

	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
)

var _ ProductManager = (*ProductService)(nil)

type ProductService struct {
	productRepo repository.ProductRepository
	variantRepo repository.ProductVariantRepository
	imageRepo   repository.ProductImageRepository
}

func NewProductService(
	productRepo repository.ProductRepository,
	variantRepo repository.ProductVariantRepository,
	imageRepo repository.ProductImageRepository,
) *ProductService {
	return &ProductService{
		productRepo: productRepo,
		variantRepo: variantRepo,
		imageRepo:   imageRepo,
	}
}

func (s *ProductService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	return s.productRepo.GetByID(ctx, id)
}

func (s *ProductService) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	return s.productRepo.GetBySlug(ctx, slug)
}

func (s *ProductService) GetBySlugAndMerchant(ctx context.Context, slug string, merchantID uuid.UUID) (*domain.Product, error) {
	return s.productRepo.GetBySlugAndMerchant(ctx, slug, merchantID)
}

func (s *ProductService) List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error) {
	return s.productRepo.List(ctx, opts)
}

func (s *ProductService) ListByMerchant(ctx context.Context, merchantID uuid.UUID, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error) {
	return s.productRepo.ListByMerchant(ctx, merchantID, opts)
}

func (s *ProductService) ListByMerchantAndStatus(ctx context.Context, merchantID uuid.UUID, status domain.ProductStatus, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error) {
	return s.productRepo.ListByMerchantAndStatus(ctx, merchantID, status, opts)
}

func (s *ProductService) ListByCategory(ctx context.Context, categoryID uuid.UUID, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error) {
	return s.productRepo.ListByCategory(ctx, categoryID, opts)
}

func (s *ProductService) Create(ctx context.Context, input CreateProductInput) (*domain.Product, error) {
	existing, _ := s.productRepo.GetBySlugAndMerchant(ctx, input.Slug, input.MerchantID)
	if existing != nil {
		return nil, domainerrors.Conflict("product with this slug already exists for this merchant")
	}

	product := &domain.Product{
		ID:          uuid.New(),
		MerchantID:  input.MerchantID,
		CategoryID:  input.CategoryID,
		Name:        input.Name,
		Slug:        input.Slug,
		Description: input.Description,
		Status:      input.Status,
		IsDigital:   input.IsDigital,
	}

	return s.productRepo.Create(ctx, product)
}

func (s *ProductService) Update(ctx context.Context, id uuid.UUID, input UpdateProductInput) (*domain.Product, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		product.Name = *input.Name
	}
	if input.CategoryID != nil {
		product.CategoryID = input.CategoryID
	}
	if input.Description != nil {
		product.Description = input.Description
	}
	if input.Status != nil {
		product.Status = *input.Status
	}
	if input.IsDigital != nil {
		product.IsDigital = *input.IsDigital
	}

	return s.productRepo.Update(ctx, product)
}

func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.imageRepo.DeleteByProduct(ctx, id); err != nil {
		return err
	}
	if err := s.variantRepo.DeleteByProduct(ctx, id); err != nil {
		return err
	}
	return s.productRepo.Delete(ctx, id)
}

var _ ProductVariantManager = (*ProductVariantService)(nil)

type ProductVariantService struct {
	variantRepo repository.ProductVariantRepository
}

func NewProductVariantService(variantRepo repository.ProductVariantRepository) *ProductVariantService {
	return &ProductVariantService{variantRepo: variantRepo}
}

func (s *ProductVariantService) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProductVariant, error) {
	return s.variantRepo.GetByID(ctx, id)
}

func (s *ProductVariantService) GetBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error) {
	return s.variantRepo.GetBySKU(ctx, sku)
}

func (s *ProductVariantService) GetByProduct(ctx context.Context, productID uuid.UUID) ([]domain.ProductVariant, error) {
	return s.variantRepo.GetByProduct(ctx, productID)
}

func (s *ProductVariantService) GetActiveByProduct(ctx context.Context, productID uuid.UUID) ([]domain.ProductVariant, error) {
	return s.variantRepo.GetActiveByProduct(ctx, productID)
}

func (s *ProductVariantService) Create(ctx context.Context, input CreateVariantInput) (*domain.ProductVariant, error) {
	existing, _ := s.variantRepo.GetBySKU(ctx, input.SKU)
	if existing != nil {
		return nil, domainerrors.Conflict("variant with this SKU already exists")
	}

	variant := &domain.ProductVariant{
		ID:                uuid.New(),
		ProductID:         input.ProductID,
		SKU:               input.SKU,
		Title:             input.Title,
		PriceCents:        input.PriceCents,
		ComparePriceCents: input.ComparePriceCents,
		Currency:          input.Currency,
		Attributes:        input.Attributes,
		IsActive:          input.IsActive,
	}

	return s.variantRepo.Create(ctx, variant)
}

func (s *ProductVariantService) Update(ctx context.Context, id uuid.UUID, input UpdateVariantInput) (*domain.ProductVariant, error) {
	variant, err := s.variantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		variant.Title = *input.Title
	}
	if input.PriceCents != nil {
		variant.PriceCents = *input.PriceCents
	}
	if input.ComparePriceCents != nil {
		variant.ComparePriceCents = input.ComparePriceCents
	}
	if input.Attributes != nil {
		variant.Attributes = input.Attributes
	}
	if input.IsActive != nil {
		variant.IsActive = *input.IsActive
	}

	return s.variantRepo.Update(ctx, variant)
}

func (s *ProductVariantService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.variantRepo.Delete(ctx, id)
}
