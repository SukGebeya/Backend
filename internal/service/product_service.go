package service

import (
	"context"
	"fmt"

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

func (s *ProductService) Browse(ctx context.Context, filter domain.ProductFilter, opts domain.ListOptions) (*domain.PaginatedResult[domain.ProductSummary], error) {
	return s.productRepo.Browse(ctx, filter, opts)
}

// Create stores a product. Slugs are unique across the marketplace so public
// product URLs are unambiguous; when none is given one is generated from the
// name (Amharic names get a random slug since they have no Latin form).
func (s *ProductService) Create(ctx context.Context, input CreateProductInput) (*domain.Product, error) {
	if input.Slug == "" {
		slug, err := s.uniqueSlug(ctx, input.Name)
		if err != nil {
			return nil, err
		}
		input.Slug = slug
	} else if existing, _ := s.productRepo.GetBySlug(ctx, input.Slug); existing != nil {
		return nil, domainerrors.Conflict("a product with this slug already exists")
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

// Delete archives the product rather than removing it: past orders reference its
// variants and must keep resolving.
func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) error {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	product.Status = domain.ProductStatusArchived
	_, err = s.productRepo.Update(ctx, product)
	return err
}

func (s *ProductService) uniqueSlug(ctx context.Context, name string) (string, error) {
	base := generateSlug(name)
	slug := base
	for i := 0; i < 5; i++ {
		existing, err := s.productRepo.GetBySlug(ctx, slug)
		if err != nil && !domainerrors.IsNotFound(err) {
			return "", err
		}
		if existing == nil {
			return slug, nil
		}
		slug = base + "-" + uuid.NewString()[:4]
	}
	return base + "-" + uuid.NewString()[:8], nil
}

func (s *ProductService) ListImages(ctx context.Context, productID uuid.UUID) ([]domain.ProductImage, error) {
	return s.imageRepo.GetByProduct(ctx, productID)
}

func (s *ProductService) AddImage(ctx context.Context, productID uuid.UUID, url string, altText *string, isPrimary bool) (*domain.ProductImage, error) {
	count, err := s.imageRepo.Count(ctx, productID)
	if err != nil {
		return nil, err
	}
	if count >= domain.MaxImagesPerProduct {
		return nil, domainerrors.BadRequest(fmt.Sprintf("a product can have at most %d images", domain.MaxImagesPerProduct))
	}
	if count == 0 {
		isPrimary = true
	}
	if isPrimary {
		if err := s.imageRepo.ClearPrimary(ctx, productID); err != nil {
			return nil, err
		}
	}
	return s.imageRepo.Create(ctx, &domain.ProductImage{
		ID:        uuid.New(),
		ProductID: productID,
		URL:       url,
		AltText:   altText,
		SortOrder: int(count),
		IsPrimary: isPrimary,
	})
}

func (s *ProductService) DeleteImage(ctx context.Context, productID, imageID uuid.UUID) error {
	images, err := s.imageRepo.GetByProduct(ctx, productID)
	if err != nil {
		return err
	}
	for _, img := range images {
		if img.ID == imageID {
			return s.imageRepo.Delete(ctx, imageID)
		}
	}
	return domainerrors.NotFound("image not found")
}

var _ ProductVariantManager = (*ProductVariantService)(nil)

type ProductVariantService struct {
	variantRepo   repository.ProductVariantRepository
	inventoryRepo repository.InventoryRepository
}

func NewProductVariantService(variantRepo repository.ProductVariantRepository, inventoryRepo repository.InventoryRepository) *ProductVariantService {
	return &ProductVariantService{variantRepo: variantRepo, inventoryRepo: inventoryRepo}
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

	created, err := s.variantRepo.Create(ctx, variant)
	if err != nil {
		return nil, err
	}

	if input.StockQuantity < 0 {
		input.StockQuantity = 0
	}
	if _, err := s.inventoryRepo.Create(ctx, &domain.Inventory{
		ID:        uuid.New(),
		VariantID: created.ID,
		Quantity:  input.StockQuantity,
	}); err != nil {
		return nil, err
	}

	return created, nil
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

// Delete deactivates the variant; order history keeps pointing at it.
func (s *ProductVariantService) Delete(ctx context.Context, id uuid.UUID) error {
	variant, err := s.variantRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	variant.IsActive = false
	_, err = s.variantRepo.Update(ctx, variant)
	return err
}
