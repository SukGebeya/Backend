package testutil

import (
	"testing"
	"time"

	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/SukGebeya/Backend/pkg/jwt"
	"github.com/SukGebeya/Backend/pkg/password"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserBuilder struct {
	user *domain.User
}

func NewUserBuilder() *UserBuilder {
	email := "test@example.com"
	phone := "+1234567890"
	return &UserBuilder{
		user: &domain.User{
			ID:           uuid.New(),
			Email:        &email,
			Phone:        &phone,
			PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3/ItB/XBG/eCknfIrqS6",
			Role:         domain.RoleBuyer,
			KYCStatus:    domain.KYCStatusPending,
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}
}

func (b *UserBuilder) WithID(id uuid.UUID) *UserBuilder {
	b.user.ID = id
	return b
}

func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.user.Email = &email
	return b
}

func (b *UserBuilder) WithPhone(phone string) *UserBuilder {
	b.user.Phone = &phone
	return b
}

func (b *UserBuilder) WithPassword(password string) *UserBuilder {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	b.user.PasswordHash = string(hash)
	return b
}

func (b *UserBuilder) WithRole(role domain.Role) *UserBuilder {
	b.user.Role = role
	return b
}

func (b *UserBuilder) WithKYCStatus(status domain.KYCStatus) *UserBuilder {
	b.user.KYCStatus = status
	return b
}

func (b *UserBuilder) Build() *domain.User {
	return b.user
}

type MerchantBuilder struct {
	merchant *domain.MerchantProfile
}

func NewMerchantBuilder() *MerchantBuilder {
	email := "merchant@test.com"
	address := "Test Address"
	return &MerchantBuilder{
		merchant: &domain.MerchantProfile{
			ID:              uuid.New(),
			ShopName:        "Test Shop",
			ShopSlug:        "test-shop-" + uuid.New().String()[:8],
			ContactPhone:    "+251911000000",
			ContactEmail:    &email,
			Address:         &address,
			City:            "Addis Ababa",
			IsVerified:      false,
			TotalSalesCents: 0,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}
}

func (b *MerchantBuilder) WithID(id uuid.UUID) *MerchantBuilder {
	b.merchant.ID = id
	return b
}

func (b *MerchantBuilder) WithUserID(userID uuid.UUID) *MerchantBuilder {
	b.merchant.UserID = userID
	return b
}

func (b *MerchantBuilder) WithShopName(name string) *MerchantBuilder {
	b.merchant.ShopName = name
	return b
}

func (b *MerchantBuilder) WithShopSlug(slug string) *MerchantBuilder {
	b.merchant.ShopSlug = slug
	return b
}

func (b *MerchantBuilder) WithVerified(verified bool) *MerchantBuilder {
	b.merchant.IsVerified = verified
	return b
}

func (b *MerchantBuilder) WithBlnkBalanceID(balanceID string) *MerchantBuilder {
	b.merchant.BlnkBalanceID = &balanceID
	return b
}

func (b *MerchantBuilder) WithTotalSales(cents int64) *MerchantBuilder {
	b.merchant.TotalSalesCents = cents
	return b
}

func (b *MerchantBuilder) Build() *domain.MerchantProfile {
	return b.merchant
}

type CategoryBuilder struct {
	category *domain.Category
}

func NewCategoryBuilder() *CategoryBuilder {
	return &CategoryBuilder{
		category: &domain.Category{
			ID:        uuid.New(),
			Name:      "Test Category",
			Slug:      "test-category",
			IsActive:  true,
			SortOrder: 1,
			CreatedAt: time.Now(),
		},
	}
}

func (b *CategoryBuilder) WithID(id uuid.UUID) *CategoryBuilder {
	b.category.ID = id
	return b
}

func (b *CategoryBuilder) WithName(name string) *CategoryBuilder {
	b.category.Name = name
	return b
}

func (b *CategoryBuilder) WithSlug(slug string) *CategoryBuilder {
	b.category.Slug = slug
	return b
}

func (b *CategoryBuilder) WithActive(active bool) *CategoryBuilder {
	b.category.IsActive = active
	return b
}

func (b *CategoryBuilder) Build() *domain.Category {
	return b.category
}

type ProductBuilder struct {
	product *domain.Product
}

func NewProductBuilder() *ProductBuilder {
	description := "Test product description"
	return &ProductBuilder{
		product: &domain.Product{
			ID:          uuid.New(),
			Name:        "Test Product",
			Slug:        "test-product-" + uuid.New().String()[:8],
			Description: &description,
			Status:      domain.ProductStatusActive,
			IsDigital:   false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
}

func (b *ProductBuilder) WithID(id uuid.UUID) *ProductBuilder {
	b.product.ID = id
	return b
}

func (b *ProductBuilder) WithMerchantID(merchantID uuid.UUID) *ProductBuilder {
	b.product.MerchantID = merchantID
	return b
}

func (b *ProductBuilder) WithCategoryID(categoryID uuid.UUID) *ProductBuilder {
	b.product.CategoryID = &categoryID
	return b
}

func (b *ProductBuilder) WithName(name string) *ProductBuilder {
	b.product.Name = name
	return b
}

func (b *ProductBuilder) WithSlug(slug string) *ProductBuilder {
	b.product.Slug = slug
	return b
}

func (b *ProductBuilder) WithStatus(status domain.ProductStatus) *ProductBuilder {
	b.product.Status = status
	return b
}

func (b *ProductBuilder) Build() *domain.Product {
	return b.product
}

type VariantBuilder struct {
	variant *domain.ProductVariant
}

func NewVariantBuilder() *VariantBuilder {
	return &VariantBuilder{
		variant: &domain.ProductVariant{
			ID:                uuid.New(),
			SKU:               "TEST-SKU-001",
			Title:             "Default",
			PriceCents:        1000,
			ComparePriceCents: nil,
			Currency:          "ETB",
			IsActive:          true,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
	}
}

func (b *VariantBuilder) WithID(id uuid.UUID) *VariantBuilder {
	b.variant.ID = id
	return b
}

func (b *VariantBuilder) WithProductID(productID uuid.UUID) *VariantBuilder {
	b.variant.ProductID = productID
	return b
}

func (b *VariantBuilder) WithSKU(sku string) *VariantBuilder {
	b.variant.SKU = sku
	return b
}

func (b *VariantBuilder) WithPrice(cents int64) *VariantBuilder {
	b.variant.PriceCents = cents
	return b
}

func (b *VariantBuilder) WithActive(active bool) *VariantBuilder {
	b.variant.IsActive = active
	return b
}

func (b *VariantBuilder) Build() *domain.ProductVariant {
	return b.variant
}

type AddressBuilder struct {
	address *domain.Address
}

func NewAddressBuilder() *AddressBuilder {
	return &AddressBuilder{
		address: &domain.Address{
			ID:        uuid.New(),
			FullName:  "Test User",
			Phone:     "+251911000000",
			Line1:     "123 Main Street",
			City:      "Addis Ababa",
			IsDefault: true,
			CreatedAt: time.Now(),
		},
	}
}

func (b *AddressBuilder) WithID(id uuid.UUID) *AddressBuilder {
	b.address.ID = id
	return b
}

func (b *AddressBuilder) WithUserID(userID uuid.UUID) *AddressBuilder {
	b.address.UserID = userID
	return b
}

func (b *AddressBuilder) WithFullName(name string) *AddressBuilder {
	b.address.FullName = name
	return b
}

func (b *AddressBuilder) WithPhone(phone string) *AddressBuilder {
	b.address.Phone = phone
	return b
}

func (b *AddressBuilder) WithLine1(line1 string) *AddressBuilder {
	b.address.Line1 = line1
	return b
}

func (b *AddressBuilder) WithCity(city string) *AddressBuilder {
	b.address.City = city
	return b
}

func (b *AddressBuilder) Build() *domain.Address {
	return b.address
}

type OrderBuilder struct {
	order *domain.Order
}

func NewOrderBuilder() *OrderBuilder {
	return &OrderBuilder{
		order: &domain.Order{
			ID:               uuid.New(),
			Status:           domain.OrderStatusPendingPayment,
			SubtotalCents:    1000,
			PlatformFeeCents: 50,
			TotalCents:       1000,
			Currency:         "ETB",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}
}

func (b *OrderBuilder) WithID(id uuid.UUID) *OrderBuilder {
	b.order.ID = id
	return b
}

func (b *OrderBuilder) WithBuyerID(buyerID uuid.UUID) *OrderBuilder {
	b.order.BuyerID = buyerID
	return b
}

func (b *OrderBuilder) WithMerchantID(merchantID uuid.UUID) *OrderBuilder {
	b.order.MerchantID = merchantID
	return b
}

func (b *OrderBuilder) WithStatus(status domain.OrderStatus) *OrderBuilder {
	b.order.Status = status
	return b
}

func (b *OrderBuilder) WithTotal(cents int64) *OrderBuilder {
	b.order.TotalCents = cents
	b.order.SubtotalCents = cents
	return b
}

func (b *OrderBuilder) WithShippingAddressID(id uuid.UUID) *OrderBuilder {
	b.order.ShippingAddressID = &id
	return b
}

func (b *OrderBuilder) WithChapaTxRef(ref string) *OrderBuilder {
	b.order.ChapaTxRef = &ref
	return b
}

func (b *OrderBuilder) WithBlnkTransactionID(id string) *OrderBuilder {
	b.order.BlnkTransactionID = &id
	return b
}

func (b *OrderBuilder) Build() *domain.Order {
	return b.order
}

type PaymentBuilder struct {
	payment *domain.Payment
}

func NewPaymentBuilder() *PaymentBuilder {
	return &PaymentBuilder{
		payment: &domain.Payment{
			ID:          uuid.New(),
			Provider:    "chapa",
			Status:      domain.PaymentStatusPending,
			AmountCents: 1000,
			Currency:    "ETB",
			CreatedAt:   time.Now(),
		},
	}
}

func (b *PaymentBuilder) WithID(id uuid.UUID) *PaymentBuilder {
	b.payment.ID = id
	return b
}

func (b *PaymentBuilder) WithOrderID(orderID uuid.UUID) *PaymentBuilder {
	b.payment.OrderID = orderID
	return b
}

func (b *PaymentBuilder) WithStatus(status domain.PaymentStatus) *PaymentBuilder {
	b.payment.Status = status
	return b
}

func (b *PaymentBuilder) WithAmount(cents int64) *PaymentBuilder {
	b.payment.AmountCents = cents
	return b
}

func (b *PaymentBuilder) Build() *domain.Payment {
	return b.payment
}

type CartBuilder struct {
	cart *domain.Cart
}

func NewCartBuilder() *CartBuilder {
	return &CartBuilder{
		cart: &domain.Cart{
			ID:        uuid.New(),
			Status:    domain.CartStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

func (b *CartBuilder) WithID(id uuid.UUID) *CartBuilder {
	b.cart.ID = id
	return b
}

func (b *CartBuilder) WithUserID(userID uuid.UUID) *CartBuilder {
	b.cart.UserID = &userID
	return b
}

func (b *CartBuilder) WithSessionID(sessionID string) *CartBuilder {
	b.cart.SessionID = &sessionID
	return b
}

func (b *CartBuilder) WithStatus(status domain.CartStatus) *CartBuilder {
	b.cart.Status = status
	return b
}

func (b *CartBuilder) Build() *domain.Cart {
	return b.cart
}

type CartItemBuilder struct {
	item *domain.CartItem
}

func NewCartItemBuilder() *CartItemBuilder {
	return &CartItemBuilder{
		item: &domain.CartItem{
			ID:                 uuid.New(),
			Quantity:           1,
			PriceSnapshotCents: 1000,
		},
	}
}

func (b *CartItemBuilder) WithID(id uuid.UUID) *CartItemBuilder {
	b.item.ID = id
	return b
}

func (b *CartItemBuilder) WithCartID(cartID uuid.UUID) *CartItemBuilder {
	b.item.CartID = cartID
	return b
}

func (b *CartItemBuilder) WithVariantID(variantID uuid.UUID) *CartItemBuilder {
	b.item.VariantID = variantID
	return b
}

func (b *CartItemBuilder) WithQuantity(qty int) *CartItemBuilder {
	b.item.Quantity = qty
	return b
}

func (b *CartItemBuilder) WithPrice(cents int64) *CartItemBuilder {
	b.item.PriceSnapshotCents = cents
	return b
}

func (b *CartItemBuilder) Build() *domain.CartItem {
	return b.item
}

type OrderItemBuilder struct {
	item *domain.OrderItem
}

func NewOrderItemBuilder() *OrderItemBuilder {
	return &OrderItemBuilder{
		item: &domain.OrderItem{
			ID:              uuid.New(),
			ProductName:     "Test Product",
			VariantTitle:    "Default",
			Quantity:        1,
			UnitPriceCents:  1000,
			TotalPriceCents: 1000,
		},
	}
}

func (b *OrderItemBuilder) WithID(id uuid.UUID) *OrderItemBuilder {
	b.item.ID = id
	return b
}

func (b *OrderItemBuilder) WithOrderID(orderID uuid.UUID) *OrderItemBuilder {
	b.item.OrderID = orderID
	return b
}

func (b *OrderItemBuilder) WithVariantID(variantID uuid.UUID) *OrderItemBuilder {
	b.item.VariantID = variantID
	return b
}

func (b *OrderItemBuilder) WithQuantity(qty int) *OrderItemBuilder {
	b.item.Quantity = qty
	return b
}

func (b *OrderItemBuilder) WithUnitPrice(cents int64) *OrderItemBuilder {
	b.item.UnitPriceCents = cents
	b.item.TotalPriceCents = cents * int64(b.item.Quantity)
	return b
}

func (b *OrderItemBuilder) Build() *domain.OrderItem {
	return b.item
}

type InventoryBuilder struct {
	inventory *domain.Inventory
}

func NewInventoryBuilder() *InventoryBuilder {
	return &InventoryBuilder{
		inventory: &domain.Inventory{
			ID:        uuid.New(),
			Quantity:  100,
			Reserved:  0,
			UpdatedAt: time.Now(),
		},
	}
}

func (b *InventoryBuilder) WithID(id uuid.UUID) *InventoryBuilder {
	b.inventory.ID = id
	return b
}

func (b *InventoryBuilder) WithVariantID(variantID uuid.UUID) *InventoryBuilder {
	b.inventory.VariantID = variantID
	return b
}

func (b *InventoryBuilder) WithQuantity(qty int) *InventoryBuilder {
	b.inventory.Quantity = qty
	return b
}

func (b *InventoryBuilder) WithReserved(reserved int) *InventoryBuilder {
	b.inventory.Reserved = reserved
	return b
}

func (b *InventoryBuilder) Build() *domain.Inventory {
	return b.inventory
}

func NewTestPasswordHasher() *password.Hasher {
	return password.NewHasher(password.DefaultConfig())
}

func NewTestJWTManager(t *testing.T) *jwt.Manager {
	t.Helper()

	accessKeys, err := jwt.GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("failed to generate access key pair: %v", err)
	}
	refreshKeys, err := jwt.GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("failed to generate refresh key pair: %v", err)
	}

	return jwt.NewManager(accessKeys, refreshKeys, 15*time.Minute, 7*24*time.Hour)
}
