package service

import (
	"context"

	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/google/uuid"
)

type Authenticator interface {
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
}

type UserManager interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.User], error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*domain.User, error)
}

type MerchantOnboarder interface {
	Onboard(ctx context.Context, input OnboardInput) (*domain.MerchantProfile, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MerchantProfile, error)
	GetBySlug(ctx context.Context, slug string) (*domain.MerchantProfile, error)
	Update(ctx context.Context, merchantID uuid.UUID, input UpdateMerchantInput) (*domain.MerchantProfile, error)
}

type CategoryManager interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Category, error)
	List(ctx context.Context) ([]domain.Category, error)
	ListByParent(ctx context.Context, parentID uuid.UUID) ([]domain.Category, error)
	ListTopLevel(ctx context.Context) ([]domain.Category, error)
	Create(ctx context.Context, input CreateCategoryInput) (*domain.Category, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateCategoryInput) (*domain.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProductManager interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Product, error)
	GetBySlugAndMerchant(ctx context.Context, slug string, merchantID uuid.UUID) (*domain.Product, error)
	List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error)
	ListByMerchant(ctx context.Context, merchantID uuid.UUID, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error)
	ListByCategory(ctx context.Context, categoryID uuid.UUID, opts domain.ListOptions) (*domain.PaginatedResult[domain.Product], error)
	Create(ctx context.Context, input CreateProductInput) (*domain.Product, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateProductInput) (*domain.Product, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProductVariantManager interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ProductVariant, error)
	GetBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error)
	GetByProduct(ctx context.Context, productID uuid.UUID) ([]domain.ProductVariant, error)
	GetActiveByProduct(ctx context.Context, productID uuid.UUID) ([]domain.ProductVariant, error)
	Create(ctx context.Context, input CreateVariantInput) (*domain.ProductVariant, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateVariantInput) (*domain.ProductVariant, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type InventoryManager interface {
	GetByVariantID(ctx context.Context, variantID uuid.UUID) (*domain.Inventory, error)
	GetByProductID(ctx context.Context, productID uuid.UUID) ([]domain.Inventory, error)
	Create(ctx context.Context, input CreateInventoryInput) (*domain.Inventory, error)
	UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) (*domain.Inventory, error)
	Reserve(ctx context.Context, variantID uuid.UUID, qty int) error
	Release(ctx context.Context, variantID uuid.UUID, qty int) error
	Commit(ctx context.Context, variantID uuid.UUID, qty int) error
}

type CreateCategoryInput struct {
	ParentID  *string
	Name      string
	Slug      string
	IconURL   *string
	IsActive  bool
	SortOrder int
}

type UpdateCategoryInput struct {
	Name      *string
	Slug      *string
	IconURL   *string
	IsActive  *bool
	SortOrder *int
}

type CreateProductInput struct {
	MerchantID  uuid.UUID
	CategoryID  *uuid.UUID
	Name        string
	Slug        string
	Description *string
	Status      domain.ProductStatus
	IsDigital   bool
}

type UpdateProductInput struct {
	Name        *string
	CategoryID  *uuid.UUID
	Description *string
	Status      *domain.ProductStatus
	IsDigital   *bool
}

type CreateVariantInput struct {
	ProductID         uuid.UUID
	SKU               string
	Title             string
	PriceCents        int64
	ComparePriceCents *int64
	Currency          string
	Attributes        map[string]any
	IsActive          bool
}

type UpdateVariantInput struct {
	Title             *string
	PriceCents        *int64
	ComparePriceCents *int64
	Attributes        map[string]any
	IsActive          *bool
}

type CreateInventoryInput struct {
	VariantID uuid.UUID
	Quantity  int
	Reserved  int
}
