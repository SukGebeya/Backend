package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type productRepository struct {
	queries *db.Queries
}

func NewProductRepository(pool *pgxpool.Pool) ProductRepository {
	return &productRepository{queries: db.New(pool)}
}

var _ ProductRepository = (*productRepository)(nil)

func (r *productRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	product, err := r.queries.GetProductByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("product not found")
		}
		return nil, domainerrors.Internal("failed to get product", err)
	}
	return dbProductToDomain(product), nil
}

func (r *productRepository) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	product, err := r.queries.GetProductBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("product not found")
		}
		return nil, domainerrors.Internal("failed to get product by slug", err)
	}
	return dbProductToDomain(product), nil
}

func (r *productRepository) GetBySlugAndMerchant(ctx context.Context, slug string, merchantID uuid.UUID) (*domain.Product, error) {
	product, err := r.queries.GetProductBySlugAndMerchant(ctx, db.GetProductBySlugAndMerchantParams{
		Slug:       slug,
		MerchantID: merchantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("product not found")
		}
		return nil, domainerrors.Internal("failed to get product", err)
	}
	return dbProductToDomain(product), nil
}

func (r *productRepository) List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error) {
	products, err := r.queries.ListProducts(ctx, db.ListProductsParams{
		PageOffset: int32(opts.Offset()),
		PageLimit:  int32(opts.Limit()),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list products", err)
	}

	items := make([]domain.Product, len(products))
	for i, p := range products {
		items[i] = *dbProductToDomain(p)
	}

	return &domain.PaginatedResult[domain.Product]{
		Items:   items,
		Total:   int64(len(items)),
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}, nil
}

func (r *productRepository) ListByMerchant(ctx context.Context, merchantID uuid.UUID, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error) {
	products, err := r.queries.ListProductsByMerchant(ctx, db.ListProductsByMerchantParams{
		MerchantID: merchantID,
		PageOffset: int32(opts.Offset()),
		PageLimit:  int32(opts.Limit()),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list products by merchant", err)
	}

	items := make([]domain.Product, len(products))
	for i, p := range products {
		items[i] = *dbProductToDomain(p)
	}

	return &domain.PaginatedResult[domain.Product]{
		Items:   items,
		Total:   int64(len(items)),
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}, nil
}

func (r *productRepository) ListByCategory(ctx context.Context, categoryID uuid.UUID, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error) {
	var pgCategoryID pgtype.UUID
	_ = pgCategoryID.Scan(categoryID.String())

	products, err := r.queries.ListProductsByCategory(ctx, db.ListProductsByCategoryParams{
		CategoryID: pgCategoryID,
		PageOffset: int32(opts.Offset()),
		PageLimit:  int32(opts.Limit()),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list products by category", err)
	}

	items := make([]domain.Product, len(products))
	for i, p := range products {
		items[i] = *dbProductToDomain(p)
	}

	return &domain.PaginatedResult[domain.Product]{
		Items:   items,
		Total:   int64(len(items)),
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}, nil
}

func (r *productRepository) ListByMerchantAndStatus(ctx context.Context, merchantID uuid.UUID, status domain.ProductStatus, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error) {
	products, err := r.queries.ListProductsByMerchantAndStatus(ctx, db.ListProductsByMerchantAndStatusParams{
		MerchantID: merchantID,
		Status:     string(status),
		PageOffset: int32(opts.Offset()),
		PageLimit:  int32(opts.Limit()),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list products by merchant and status", err)
	}

	items := make([]domain.Product, len(products))
	for i, p := range products {
		items[i] = *dbProductToDomain(p)
	}

	return &domain.PaginatedResult[domain.Product]{
		Items:   items,
		Total:   int64(len(items)),
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}, nil
}

func (r *productRepository) Create(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	var pgCategoryID pgtype.UUID
	if product.CategoryID != nil {
		_ = pgCategoryID.Scan(product.CategoryID.String())
	}

	created, err := r.queries.CreateProduct(ctx, db.CreateProductParams{
		ID:          product.ID,
		MerchantID:  product.MerchantID,
		CategoryID:  pgCategoryID,
		Name:        product.Name,
		Slug:        product.Slug,
		Description: product.Description,
		Status:      string(product.Status),
		IsDigital:   product.IsDigital,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create product", err)
	}
	return dbProductToDomain(created), nil
}

func (r *productRepository) Update(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	var pgCategoryID pgtype.UUID
	if product.CategoryID != nil {
		_ = pgCategoryID.Scan(product.CategoryID.String())
	}

	updated, err := r.queries.UpdateProduct(ctx, db.UpdateProductParams{
		ID:          product.ID,
		Name:        product.Name,
		CategoryID:  pgCategoryID,
		Description: product.Description,
		Status:      string(product.Status),
		IsDigital:   product.IsDigital,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("product not found")
		}
		return nil, domainerrors.Internal("failed to update product", err)
	}
	return dbProductToDomain(updated), nil
}

func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteProduct(ctx, id)
}

func (r *productRepository) CountByMerchant(ctx context.Context, merchantID uuid.UUID) (int64, error) {
	return r.queries.CountProductsByMerchant(ctx, merchantID)
}

func dbProductToDomain(p db.Product) *domain.Product {
	var categoryID *uuid.UUID
	if p.CategoryID.Valid {
		cid, _ := uuid.FromBytes(p.CategoryID.Bytes[:])
		categoryID = &cid
	}

	return &domain.Product{
		ID:          p.ID,
		MerchantID:  p.MerchantID,
		CategoryID:  categoryID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Status:      domain.ProductStatus(p.Status),
		IsDigital:   p.IsDigital,
		CreatedAt:   p.CreatedAt.Time,
		UpdatedAt:   p.UpdatedAt.Time,
	}
}

type productImageRepository struct {
	queries *db.Queries
}

func NewProductImageRepository(pool *pgxpool.Pool) ProductImageRepository {
	return &productImageRepository{queries: db.New(pool)}
}

var _ ProductImageRepository = (*productImageRepository)(nil)

func (r *productImageRepository) GetByProduct(ctx context.Context, productID uuid.UUID) ([]domain.ProductImage, error) {
	images, err := r.queries.GetProductImages(ctx, productID)
	if err != nil {
		return nil, domainerrors.Internal("failed to get product images", err)
	}

	result := make([]domain.ProductImage, len(images))
	for i, img := range images {
		result[i] = *dbProductImageToDomain(img)
	}
	return result, nil
}

func (r *productImageRepository) GetPrimary(ctx context.Context, productID uuid.UUID) (*domain.ProductImage, error) {
	image, err := r.queries.GetProductPrimaryImage(ctx, productID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("primary image not found")
		}
		return nil, domainerrors.Internal("failed to get primary image", err)
	}
	return dbProductImageToDomain(image), nil
}

func (r *productImageRepository) Create(ctx context.Context, image *domain.ProductImage) (*domain.ProductImage, error) {
	created, err := r.queries.CreateProductImage(ctx, db.CreateProductImageParams{
		ID:        image.ID,
		ProductID: image.ProductID,
		Url:       image.URL,
		AltText:   image.AltText,
		SortOrder: int32(image.SortOrder),
		IsPrimary: image.IsPrimary,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create product image", err)
	}
	return dbProductImageToDomain(created), nil
}

func (r *productImageRepository) Update(ctx context.Context, image *domain.ProductImage) (*domain.ProductImage, error) {
	updated, err := r.queries.UpdateProductImage(ctx, db.UpdateProductImageParams{
		ID:        image.ID,
		Url:       image.URL,
		AltText:   image.AltText,
		SortOrder: int32(image.SortOrder),
		IsPrimary: image.IsPrimary,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("product image not found")
		}
		return nil, domainerrors.Internal("failed to update product image", err)
	}
	return dbProductImageToDomain(updated), nil
}

func (r *productImageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteProductImage(ctx, id)
}

func (r *productImageRepository) DeleteByProduct(ctx context.Context, productID uuid.UUID) error {
	return r.queries.DeleteProductImages(ctx, productID)
}

func dbProductImageToDomain(img db.ProductImage) *domain.ProductImage {
	return &domain.ProductImage{
		ID:        img.ID,
		ProductID: img.ProductID,
		URL:       img.Url,
		AltText:   img.AltText,
		SortOrder: int(img.SortOrder),
		IsPrimary: img.IsPrimary,
	}
}

type productVariantRepository struct {
	queries *db.Queries
}

func NewProductVariantRepository(pool *pgxpool.Pool) ProductVariantRepository {
	return &productVariantRepository{queries: db.New(pool)}
}

var _ ProductVariantRepository = (*productVariantRepository)(nil)

func (r *productVariantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProductVariant, error) {
	variant, err := r.queries.GetVariantByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("variant not found")
		}
		return nil, domainerrors.Internal("failed to get variant", err)
	}
	return dbProductVariantToDomain(variant), nil
}

func (r *productVariantRepository) GetBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error) {
	variant, err := r.queries.GetVariantBySKU(ctx, sku)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("variant not found")
		}
		return nil, domainerrors.Internal("failed to get variant by SKU", err)
	}
	return dbProductVariantToDomain(variant), nil
}

func (r *productVariantRepository) GetByProduct(ctx context.Context, productID uuid.UUID) ([]domain.ProductVariant, error) {
	variants, err := r.queries.GetVariantsByProduct(ctx, productID)
	if err != nil {
		return nil, domainerrors.Internal("failed to get variants", err)
	}

	result := make([]domain.ProductVariant, len(variants))
	for i, v := range variants {
		result[i] = *dbProductVariantToDomain(v)
	}
	return result, nil
}

func (r *productVariantRepository) GetActiveByProduct(ctx context.Context, productID uuid.UUID) ([]domain.ProductVariant, error) {
	variants, err := r.queries.GetActiveVariantsByProduct(ctx, productID)
	if err != nil {
		return nil, domainerrors.Internal("failed to get active variants", err)
	}

	result := make([]domain.ProductVariant, len(variants))
	for i, v := range variants {
		result[i] = *dbProductVariantToDomain(v)
	}
	return result, nil
}

func (r *productVariantRepository) Create(ctx context.Context, variant *domain.ProductVariant) (*domain.ProductVariant, error) {
	attrsJSON, err := json.Marshal(variant.Attributes)
	if err != nil {
		return nil, domainerrors.Internal("failed to marshal attributes", err)
	}

	created, err := r.queries.CreateVariant(ctx, db.CreateVariantParams{
		ID:                variant.ID,
		ProductID:         variant.ProductID,
		Sku:               variant.SKU,
		Title:             variant.Title,
		PriceCents:        variant.PriceCents,
		ComparePriceCents: variant.ComparePriceCents,
		Currency:          variant.Currency,
		Attributes:        attrsJSON,
		IsActive:          variant.IsActive,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create variant", err)
	}
	return dbProductVariantToDomain(created), nil
}

func (r *productVariantRepository) Update(ctx context.Context, variant *domain.ProductVariant) (*domain.ProductVariant, error) {
	attrsJSON, err := json.Marshal(variant.Attributes)
	if err != nil {
		return nil, domainerrors.Internal("failed to marshal attributes", err)
	}

	updated, err := r.queries.UpdateVariant(ctx, db.UpdateVariantParams{
		ID:                variant.ID,
		Title:             variant.Title,
		PriceCents:        variant.PriceCents,
		ComparePriceCents: variant.ComparePriceCents,
		Attributes:        attrsJSON,
		IsActive:          variant.IsActive,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("variant not found")
		}
		return nil, domainerrors.Internal("failed to update variant", err)
	}
	return dbProductVariantToDomain(updated), nil
}

func (r *productVariantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteVariant(ctx, id)
}

func (r *productVariantRepository) DeleteByProduct(ctx context.Context, productID uuid.UUID) error {
	return r.queries.DeleteVariantsByProduct(ctx, productID)
}

func dbProductVariantToDomain(v db.ProductVariant) *domain.ProductVariant {
	var attrs map[string]any
	if len(v.Attributes) > 0 {
		_ = json.Unmarshal(v.Attributes, &attrs)
	}

	return &domain.ProductVariant{
		ID:                v.ID,
		ProductID:         v.ProductID,
		SKU:               v.Sku,
		Title:             v.Title,
		PriceCents:        v.PriceCents,
		ComparePriceCents: v.ComparePriceCents,
		Currency:          v.Currency,
		Attributes:        attrs,
		IsActive:          v.IsActive,
		CreatedAt:         v.CreatedAt.Time,
		UpdatedAt:         v.UpdatedAt.Time,
	}
}
