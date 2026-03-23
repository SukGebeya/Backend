package repository

import (
	"context"

	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/google/uuid"
)

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByPhone(ctx context.Context, phone string) (*domain.User, error)
	GetByEmailOrPhone(ctx context.Context, identifier string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.User], error)
	Count(ctx context.Context) (int64, error)
}

type RefreshTokenRepository interface {
	Store(ctx context.Context, userID uuid.UUID, tokenID string, ttl int64) error
	Get(ctx context.Context, userID uuid.UUID, tokenID string) (bool, error)
	Delete(ctx context.Context, userID uuid.UUID, tokenID string) error
	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
}

type MerchantRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.MerchantProfile, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MerchantProfile, error)
	GetBySlug(ctx context.Context, slug string) (*domain.MerchantProfile, error)
	Create(ctx context.Context, merchant *domain.MerchantProfile) (*domain.MerchantProfile, error)
	Update(ctx context.Context, merchant *domain.MerchantProfile) (*domain.MerchantProfile, error)
	UpdateBlnkWallet(ctx context.Context, id uuid.UUID, balanceID, ledgerID string) (*domain.MerchantProfile, error)
	List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.MerchantProfile], error)
}

type CategoryRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Category, error)
	List(ctx context.Context) ([]domain.Category, error)
	ListByParent(ctx context.Context, parentID uuid.UUID) ([]domain.Category, error)
	ListTopLevel(ctx context.Context) ([]domain.Category, error)
	Create(ctx context.Context, category *domain.Category) (*domain.Category, error)
	Update(ctx context.Context, category *domain.Category) (*domain.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProductRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Product, error)
	GetBySlugAndMerchant(ctx context.Context, slug string, merchantID uuid.UUID) (*domain.Product, error)
	List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error)
	ListByMerchant(ctx context.Context, merchantID uuid.UUID, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error)
	ListByCategory(ctx context.Context, categoryID uuid.UUID, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error)
	ListByMerchantAndStatus(ctx context.Context, merchantID uuid.UUID, status domain.ProductStatus, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error)
	Create(ctx context.Context, product *domain.Product) (*domain.Product, error)
	Update(ctx context.Context, product *domain.Product) (*domain.Product, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountByMerchant(ctx context.Context, merchantID uuid.UUID) (int64, error)
}

type ProductImageRepository interface {
	GetByProduct(ctx context.Context, productID uuid.UUID) ([]domain.ProductImage, error)
	GetPrimary(ctx context.Context, productID uuid.UUID) (*domain.ProductImage, error)
	Create(ctx context.Context, image *domain.ProductImage) (*domain.ProductImage, error)
	Update(ctx context.Context, image *domain.ProductImage) (*domain.ProductImage, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByProduct(ctx context.Context, productID uuid.UUID) error
}

type ProductVariantRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ProductVariant, error)
	GetBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error)
	GetByProduct(ctx context.Context, productID uuid.UUID) ([]domain.ProductVariant, error)
	GetActiveByProduct(ctx context.Context, productID uuid.UUID) ([]domain.ProductVariant, error)
	Create(ctx context.Context, variant *domain.ProductVariant) (*domain.ProductVariant, error)
	Update(ctx context.Context, variant *domain.ProductVariant) (*domain.ProductVariant, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByProduct(ctx context.Context, productID uuid.UUID) error
}

type InventoryRepository interface {
	GetByVariantID(ctx context.Context, variantID uuid.UUID) (*domain.Inventory, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Inventory, error)
	GetByProductID(ctx context.Context, productID uuid.UUID) ([]domain.Inventory, error)
	Create(ctx context.Context, inventory *domain.Inventory) (*domain.Inventory, error)
	UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) (*domain.Inventory, error)
	Reserve(ctx context.Context, variantID uuid.UUID, qty int) error
	Release(ctx context.Context, variantID uuid.UUID, qty int) error
	Commit(ctx context.Context, variantID uuid.UUID, qty int) error
}

type RateLimitRepository interface {
	Increment(ctx context.Context, key string, window int64) (int64, error)
	GetTTL(ctx context.Context, key string) (int64, error)
	Reset(ctx context.Context, key string) error
}
