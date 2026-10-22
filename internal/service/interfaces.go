package service

import (
	"context"
	"time"

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
	UpdateKYCStatus(ctx context.Context, userID uuid.UUID, kycStatus domain.KYCStatus) (*domain.User, error)
}

type MerchantOnboarder interface {
	Onboard(ctx context.Context, input OnboardInput) (*domain.MerchantProfile, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MerchantProfile, error)
	GetBySlug(ctx context.Context, slug string) (*domain.MerchantProfile, error)
	Update(ctx context.Context, merchantID uuid.UUID, input UpdateMerchantInput) (*domain.MerchantProfile, error)
	Verify(ctx context.Context, merchantID uuid.UUID, isVerified bool) (*domain.MerchantProfile, error)
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
	SetStock(ctx context.Context, variantID uuid.UUID, quantity int) (*domain.Inventory, error)
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
	StockQuantity     int
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

type AddressManager interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Address, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Address, error)
	GetDefault(ctx context.Context, userID uuid.UUID) (*domain.Address, error)
	Create(ctx context.Context, input CreateAddressInput) (*domain.Address, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateAddressInput) (*domain.Address, error)
	SetDefault(ctx context.Context, userID, addressID uuid.UUID) (*domain.Address, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type CreateAddressInput struct {
	UserID    uuid.UUID
	FullName  string
	Phone     string
	Line1     string
	Line2     *string
	City      string
	Subcity   *string
	Woreda    *string
	IsDefault bool
}

type UpdateAddressInput struct {
	FullName  *string
	Phone     *string
	Line1     *string
	Line2     *string
	City      *string
	Subcity   *string
	Woreda    *string
	IsDefault *bool
}

type CartManager interface {
	GetOrCreateCart(ctx context.Context, userID *uuid.UUID, sessionID *string) (*domain.Cart, error)
	GetCart(ctx context.Context, cartID uuid.UUID) (*domain.Cart, error)
	AddItem(ctx context.Context, cartID uuid.UUID, variantID uuid.UUID, quantity int, priceCents int64) (*domain.CartItem, error)
	UpdateItemQuantity(ctx context.Context, cartID, itemID uuid.UUID, quantity int) (*domain.CartItem, error)
	UpdateItemQuantityByVariant(ctx context.Context, cartID, variantID uuid.UUID, quantity int) (*domain.CartItem, error)
	RemoveItem(ctx context.Context, cartID, itemID uuid.UUID) error
	RemoveItemByVariant(ctx context.Context, cartID, variantID uuid.UUID) error
	ClearCart(ctx context.Context, cartID uuid.UUID) error
	MergeCart(ctx context.Context, userID uuid.UUID, sessionID string) (*domain.Cart, error)
	GetCartWithItems(ctx context.Context, cartID uuid.UUID) (*domain.CartWithDetails, error)
}

type SessionManager interface {
	CreateSession(ctx context.Context) (string, error)
	ValidateSession(ctx context.Context, sessionID string) (bool, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

type JobScheduler interface {
	ScheduleCartExpiry(ctx context.Context, cartID uuid.UUID, expiresAt time.Time) error
}

type PaymentManager interface {
	ProcessPayment(ctx context.Context, txRef string) error
	ExpireUnpaidOrder(ctx context.Context, orderID uuid.UUID) error
	RefundOrder(ctx context.Context, orderID uuid.UUID) error
}

type OrderManager interface {
	GetByID(ctx context.Context, orderID uuid.UUID) (*domain.Order, error)
	GetByChapaTxRef(ctx context.Context, chapaTxRef string) (*domain.Order, error)
	ListByBuyerID(ctx context.Context, buyerID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.Order], error)
	ListByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.Order], error)
	ListByStatus(ctx context.Context, status domain.OrderStatus, page, pageSize int) (*domain.PaginatedResult[domain.Order], error)
	Create(ctx context.Context, input CreateOrderInput) (*domain.Order, error)
	UpdateStatus(ctx context.Context, orderID uuid.UUID, status domain.OrderStatus) (*domain.Order, error)
	UpdateBlnkTransaction(ctx context.Context, orderID uuid.UUID, blnkTxID string) (*domain.Order, error)
	GetOrderWithItems(ctx context.Context, orderID uuid.UUID) (*domain.OrderWithItems, error)
	GetOrderWithDetails(ctx context.Context, orderID uuid.UUID) (*OrderWithDetails, error)
	ConfirmDelivery(ctx context.Context, orderID uuid.UUID) (*domain.Order, error)
	MarkOrderShipped(ctx context.Context, input MarkOrderShippedInput) (*domain.Order, error)
	CancelOrder(ctx context.Context, orderID uuid.UUID, cancelledBy string) (*domain.Order, error)
}

type CheckoutManager interface {
	Checkout(ctx context.Context, input CheckoutInput) (*CheckoutResult, error)
}

type CreateOrderInput struct {
	BuyerID           uuid.UUID
	MerchantID        uuid.UUID
	ShippingAddressID *uuid.UUID
	Items             []CreateOrderItemInput
	SubtotalCents     int64
	PlatformFeeCents  int64
	TotalCents        int64
	Currency          string
	Notes             *string
	ChapaTxRef        string
}

type CreateOrderItemInput struct {
	VariantID       uuid.UUID
	ProductName     string
	VariantTitle    string
	Quantity        int
	UnitPriceCents  int64
	TotalPriceCents int64
}

type CheckoutInput struct {
	UserID            uuid.UUID
	ShippingAddressID *uuid.UUID
	Notes             *string
}

type CheckoutResult struct {
	OrderID    uuid.UUID
	TotalCents int64
	Currency   string
	PaymentURL string
	ChapaTxRef string
}

type MarkOrderShippedInput struct {
	OrderID           uuid.UUID
	Carrier           *string
	TrackingNumber    *string
	EstimatedDelivery *time.Time
}

type OrderWithDetails struct {
	Order           domain.Order
	Items           []domain.OrderItem
	Shipment        *domain.Shipment
	ShippingAddress *domain.Address
}

type WalletManager interface {
	GetBalance(ctx context.Context, merchantID uuid.UUID) (*WalletBalanceResponse, error)
	GetTransactionHistory(ctx context.Context, merchantID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.WalletTransaction], error)
	RequestPayout(ctx context.Context, merchantID uuid.UUID, input domain.RequestPayoutInput) (*domain.Payout, error)
	ListPayoutsByMerchant(ctx context.Context, merchantID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.Payout], error)
}

type WalletBalanceResponse struct {
	BalanceCents       int64  `json:"balance_cents"`
	PendingPayoutCents int64  `json:"pending_payout_cents"`
	AvailableCents     int64  `json:"available_cents"`
	Currency           string `json:"currency"`
}

type AdminPayoutManager interface {
	ListPayouts(ctx context.Context, status *domain.PayoutStatus, page, pageSize int) (*domain.PaginatedResult[domain.Payout], error)
	ApprovePayout(ctx context.Context, payoutID uuid.UUID) (*domain.Payout, error)
	RejectPayout(ctx context.Context, payoutID uuid.UUID, reason string) (*domain.Payout, error)
}
