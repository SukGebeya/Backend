package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/mocks"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/testutil"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type harness struct {
	ctx   context.Context
	redis *goredis.Client

	users      repository.UserRepository
	merchants  repository.MerchantRepository
	products   repository.ProductRepository
	variants   repository.ProductVariantRepository
	inventory  repository.InventoryRepository
	carts      repository.CartRepository
	cartItems  repository.CartItemRepository
	orders     repository.OrderRepository
	payments   repository.PaymentRepository
	addresses  repository.AddressRepository
	shipments  repository.ShipmentRepository
	walletTxns repository.WalletTransactionRepository
	payouts    repository.PayoutRepository

	blnk     *mocks.MockBlnkClient
	chapa    *mocks.MockChapaClient
	enqueuer *jobs.NoopEnqueuer

	cart          *CartService
	checkout      *CheckoutService
	payment       *PaymentService
	order         *OrderService
	wallet        *WalletService
	inventorySvc  *InventoryService
	reconcile     *ReconciliationService
	chapaPayments map[string]int64
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	pool := testutil.NewTestDatabase(t)
	redisClient := testutil.NewTestRedis(t)

	h := &harness{
		ctx:           context.Background(),
		redis:         redisClient,
		users:         repository.NewUserRepository(pool),
		merchants:     repository.NewMerchantRepository(pool),
		products:      repository.NewProductRepository(pool),
		variants:      repository.NewProductVariantRepository(pool),
		inventory:     repository.NewInventoryRepository(pool),
		carts:         repository.NewCartRepository(pool),
		cartItems:     repository.NewCartItemRepository(pool),
		orders:        repository.NewOrderRepository(pool),
		payments:      repository.NewPaymentRepository(pool),
		addresses:     repository.NewAddressRepository(pool),
		shipments:     repository.NewShipmentRepository(pool),
		walletTxns:    repository.NewWalletTransactionRepository(pool),
		payouts:       repository.NewPayoutRepository(pool),
		blnk:          mocks.NewMockBlnkClient(),
		chapa:         mocks.NewMockChapaClient(),
		enqueuer:      &jobs.NoopEnqueuer{},
		chapaPayments: make(map[string]int64),
	}

	h.chapa.InitializePaymentFunc = func(ctx context.Context, req chapa.InitializePaymentRequest) (*chapa.InitializePaymentResponse, error) {
		cents, err := chapa.ParseAmountCents(req.Amount)
		if err != nil {
			return nil, err
		}
		h.chapaPayments[req.TxRef] = cents
		return &chapa.InitializePaymentResponse{Status: chapa.StatusSuccess, Data: chapa.InitializePaymentData{CheckoutURL: "https://checkout.chapa.co/" + req.TxRef, TxRef: req.TxRef}}, nil
	}
	h.setChapaVerify(chapa.StatusSuccess, -1)

	txManager := repository.NewTransactionManager(pool)
	sessions := repository.NewSessionRepository(redisClient)

	h.cart = NewCartService(h.carts, h.cartItems, sessions, h.variants, h.inventory, h.enqueuer)
	h.checkout = NewCheckoutService(h.carts, h.cartItems, h.orders, h.payments, h.addresses, h.inventory, h.products, h.variants, h.users, h.merchants, h.chapa, txManager, h.enqueuer,
		CheckoutConfig{FeePercent: 5, PaymentTimeout: time.Hour, RequirePhoneVerification: true})
	h.payment = NewPaymentService(PaymentDeps{
		Orders: h.orders, Payments: h.payments, Inventory: h.inventory, Merchants: h.merchants, WalletTxns: h.walletTxns,
		Blnk: h.blnk, Chapa: h.chapa, TxManager: txManager, Enqueuer: h.enqueuer,
	})
	h.order = NewOrderService(h.orders, h.merchants, h.shipments, h.walletTxns, h.payments, h.blnk, h.inventory, txManager, h.enqueuer).
		WithAddresses(h.addresses)
	h.wallet = NewWalletService(h.merchants, h.users, h.walletTxns, h.payouts, h.blnk, h.chapa, txManager, h.enqueuer).
		WithPolicy(PayoutPolicy{MinCents: 100, DailyLimitCents: 1_000_000, RequireKYC: true})
	h.inventorySvc = NewInventoryService(h.inventory)
	h.reconcile = NewReconciliationService(h.merchants, h.walletTxns, h.orders, h.blnk)
	return h
}

// setChapaVerify makes Chapa verification return the given status. An amount of
// -1 echoes the amount the payment was initialized with.
func (h *harness) setChapaVerify(status string, amountCents int64) {
	h.chapa.VerifyPaymentFunc = func(ctx context.Context, txRef string) (*chapa.VerifyPaymentResponse, error) {
		amount := amountCents
		if amount < 0 {
			amount = h.chapaPayments[txRef]
		}
		return &chapa.VerifyPaymentResponse{
			Status: chapa.StatusSuccess,
			Data: chapa.VerifyPaymentData{TxRef: txRef, Status: status, Amount: json.Number(chapa.FormatAmount(amount)),
				Charge: json.Number(chapa.FormatAmount(chapa.FakeChargeCents(amount))), Currency: "ETB", Mode: chapa.ModeTest},
		}, nil
	}
}

type fixture struct {
	buyer    *domain.User
	seller   *domain.User
	merchant *domain.MerchantProfile
	product  *domain.Product
	variant  *domain.ProductVariant
	address  *domain.Address
}

func (h *harness) seed(t *testing.T, priceCents int64, stock int) fixture {
	t.Helper()

	buyer := testutil.NewUserBuilder().WithEmail("buyer-" + uuid.NewString()[:8] + "@test.com").WithPhone(randomPhone("9")).Build()
	buyer, err := h.users.Create(h.ctx, buyer)
	require.NoError(t, err)
	require.NoError(t, h.users.MarkPhoneVerified(h.ctx, buyer.ID))
	buyer, err = h.users.GetByID(h.ctx, buyer.ID)
	require.NoError(t, err)

	seller := testutil.NewUserBuilder().WithEmail("seller-" + uuid.NewString()[:8] + "@test.com").WithPhone(randomPhone("7")).
		WithRole(domain.RoleMerchant).WithKYCStatus(domain.KYCStatusVerified).Build()
	seller, err = h.users.Create(h.ctx, seller)
	require.NoError(t, err)

	merchant, err := h.merchants.Create(h.ctx, testutil.NewMerchantBuilder().WithUserID(seller.ID).WithBlnkBalanceID("bln_merchant_"+uuid.NewString()[:8]).Build())
	require.NoError(t, err)
	merchant.IsVerified = true
	merchant, err = h.merchants.Update(h.ctx, merchant)
	require.NoError(t, err)

	address, err := h.addresses.Create(h.ctx, testutil.NewAddressBuilder().WithUserID(buyer.ID).Build())
	require.NoError(t, err)
	if merchant.BlnkBalanceID == nil {
		merchant, err = h.merchants.UpdateBlnkWallet(h.ctx, merchant.ID, "bln_merchant_"+uuid.NewString()[:8], "ldg_test")
		require.NoError(t, err)
	}

	product, err := h.products.Create(h.ctx, testutil.NewProductBuilder().WithMerchantID(merchant.ID).WithName("Habesha Kemis").Build())
	require.NoError(t, err)

	variant, err := h.variants.Create(h.ctx, testutil.NewVariantBuilder().WithProductID(product.ID).WithSKU("SKU-"+uuid.NewString()[:8]).WithPrice(priceCents).Build())
	require.NoError(t, err)

	_, err = h.inventory.Create(h.ctx, &domain.Inventory{ID: uuid.New(), VariantID: variant.ID, Quantity: stock})
	require.NoError(t, err)

	return fixture{buyer: buyer, seller: seller, merchant: merchant, product: product, variant: variant, address: address}
}

func randomPhone(prefix string) string {
	digits := strings.ReplaceAll(uuid.NewString(), "-", "")
	out := []byte("+251" + prefix)
	for _, c := range digits {
		if len(out) == 13 {
			break
		}
		out = append(out, byte('0'+int(c)%10))
	}
	return string(out)
}

func (h *harness) addToCart(t *testing.T, userID, variantID uuid.UUID, qty int, priceCents int64) *domain.Cart {
	t.Helper()
	cart, err := h.cart.GetOrCreateCart(h.ctx, &userID, nil)
	require.NoError(t, err)
	_, err = h.cart.AddItem(h.ctx, cart.ID, variantID, qty, priceCents)
	require.NoError(t, err)
	return cart
}

func (h *harness) stock(t *testing.T, variantID uuid.UUID) (quantity, reserved int) {
	t.Helper()
	inv, err := h.inventory.GetByVariantID(h.ctx, variantID)
	require.NoError(t, err)
	return inv.Quantity, inv.Reserved
}

func (h *harness) checkoutOrder(t *testing.T, f fixture, qty int) *domain.Order {
	t.Helper()
	h.addToCart(t, f.buyer.ID, f.variant.ID, qty, f.variant.PriceCents)
	result, err := h.checkout.Checkout(h.ctx, CheckoutInput{UserID: f.buyer.ID, ShippingAddressID: &f.address.ID})
	require.NoError(t, err)
	order, err := h.orders.GetByID(h.ctx, result.OrderID)
	require.NoError(t, err)
	return order
}

func (h *harness) paidOrder(t *testing.T, f fixture, qty int) *domain.Order {
	t.Helper()
	order := h.checkoutOrder(t, f, qty)
	require.NoError(t, h.payment.ProcessPayment(h.ctx, *order.ChapaTxRef))
	order, err := h.orders.GetByID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusPaid, order.Status)
	return order
}

func (h *harness) enqueued(kind string) int {
	n := 0
	for _, k := range h.enqueuer.Kinds() {
		if k == kind {
			n++
		}
	}
	return n
}
