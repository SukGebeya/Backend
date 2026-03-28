package repository

import (
	"context"
	"time"

	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	WithTx(tx pgx.Tx) MerchantRepository
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
	WithTx(tx pgx.Tx) InventoryRepository
}

type RateLimitRepository interface {
	Increment(ctx context.Context, key string, window int64) (int64, error)
	GetTTL(ctx context.Context, key string) (int64, error)
	Reset(ctx context.Context, key string) error
}

type AddressRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Address, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Address, error)
	GetDefault(ctx context.Context, userID uuid.UUID) (*domain.Address, error)
	Create(ctx context.Context, address *domain.Address) (*domain.Address, error)
	Update(ctx context.Context, address *domain.Address) (*domain.Address, error)
	SetDefault(ctx context.Context, userID, addressID uuid.UUID) (*domain.Address, error)
	ClearDefaults(ctx context.Context, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

type CartRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Cart, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Cart, error)
	GetBySessionID(ctx context.Context, sessionID string) (*domain.Cart, error)
	Create(ctx context.Context, cart *domain.Cart) (*domain.Cart, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.CartStatus) (*domain.Cart, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBySessionID(ctx context.Context, sessionID string) error
	WithTx(tx pgx.Tx) CartRepository
}

type CartItemRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.CartItem, error)
	GetByCartAndVariant(ctx context.Context, cartID, variantID uuid.UUID) (*domain.CartItem, error)
	ListByCart(ctx context.Context, cartID uuid.UUID) ([]domain.CartItem, error)
	Create(ctx context.Context, item *domain.CartItem) (*domain.CartItem, error)
	Upsert(ctx context.Context, item *domain.CartItem) (*domain.CartItem, error)
	UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) (*domain.CartItem, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByCartAndVariant(ctx context.Context, cartID, variantID uuid.UUID) error
	ClearCart(ctx context.Context, cartID uuid.UUID) error
	CountByCart(ctx context.Context, cartID uuid.UUID) (int64, error)
	CountDistinctVariants(ctx context.Context, cartID uuid.UUID) (int, error)
	MergeItems(ctx context.Context, sourceCartID, targetCartID uuid.UUID) error
	WithTx(tx pgx.Tx) CartItemRepository
}

type SessionRepository interface {
	Create(ctx context.Context, sessionID string, ttl int64) error
	Get(ctx context.Context, sessionID string) (bool, error)
	Delete(ctx context.Context, sessionID string) error
	AssociateUser(ctx context.Context, sessionID string, userID uuid.UUID) error
}

type OrderRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	GetByChapaTxRef(ctx context.Context, chapaTxRef string) (*domain.Order, error)
	ListByBuyerID(ctx context.Context, buyerID uuid.UUID, page, pageSize int) ([]domain.Order, error)
	ListByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]domain.Order, error)
	ListByStatus(ctx context.Context, status domain.OrderStatus, page, pageSize int) ([]domain.Order, error)
	Create(ctx context.Context, order *domain.Order) (*domain.Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (*domain.Order, error)
	UpdateBlnkTransaction(ctx context.Context, id uuid.UUID, blnkTxID string) (*domain.Order, error)
	CountByBuyerID(ctx context.Context, buyerID uuid.UUID) (int64, error)
	CountByMerchantID(ctx context.Context, merchantID uuid.UUID) (int64, error)
	GetItemByID(ctx context.Context, id uuid.UUID) (*domain.OrderItem, error)
	ListItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error)
	CreateItem(ctx context.Context, item *domain.OrderItem) (*domain.OrderItem, error)
	WithTx(tx pgx.Tx) OrderRepository
}

type PaymentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Payment, error)
	GetByProviderTxRef(ctx context.Context, providerTxRef string) (*domain.Payment, error)
	ListByStatus(ctx context.Context, status domain.PaymentStatus, page, pageSize int) ([]domain.Payment, error)
	Create(ctx context.Context, payment *domain.Payment) (*domain.Payment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.PaymentStatus) (*domain.Payment, error)
	UpdateProviderResponse(ctx context.Context, id uuid.UUID, providerResponse map[string]any) (*domain.Payment, error)
	UpdateBlnkTransaction(ctx context.Context, id uuid.UUID, blnkTxID string) (*domain.Payment, error)
	WithTx(tx pgx.Tx) PaymentRepository
}

type ShipmentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Shipment, error)
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Shipment, error)
	Create(ctx context.Context, shipment *domain.Shipment) (*domain.Shipment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ShipmentStatus) (*domain.Shipment, error)
	UpdateShipped(ctx context.Context, id uuid.UUID, carrier, trackingNumber *string, estimatedDelivery *time.Time) (*domain.Shipment, error)
	UpdateDelivered(ctx context.Context, id uuid.UUID) (*domain.Shipment, error)
	WithTx(tx pgx.Tx) ShipmentRepository
}

type WalletTransactionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.WalletTransaction, error)
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.WalletTransaction, error)
	ListByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]domain.WalletTransaction, error)
	Create(ctx context.Context, txn *domain.WalletTransaction) (*domain.WalletTransaction, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.WalletTransactionStatus) (*domain.WalletTransaction, error)
	CountByMerchantID(ctx context.Context, merchantID uuid.UUID) (int64, error)
	WithTx(tx pgx.Tx) WalletTransactionRepository
}

type PayoutRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Payout, error)
	ListByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]domain.Payout, error)
	ListByStatus(ctx context.Context, status domain.PayoutStatus, page, pageSize int) ([]domain.Payout, error)
	ListAll(ctx context.Context, page, pageSize int) ([]domain.Payout, error)
	Create(ctx context.Context, payout *domain.Payout) (*domain.Payout, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.PayoutStatus, blnkTxID *string, failureReason *string) (*domain.Payout, error)
	CountByMerchantID(ctx context.Context, merchantID uuid.UUID) (int64, error)
	CountByStatus(ctx context.Context, status domain.PayoutStatus) (int64, error)
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl int64) error
}
