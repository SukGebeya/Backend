package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/phone"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

var _ CheckoutManager = (*CheckoutService)(nil)

type CheckoutService struct {
	cartRepo       repository.CartRepository
	cartItemRepo   repository.CartItemRepository
	orderRepo      repository.OrderRepository
	paymentRepo    repository.PaymentRepository
	addressRepo    repository.AddressRepository
	inventoryRepo  repository.InventoryRepository
	productRepo    repository.ProductRepository
	variantRepo    repository.ProductVariantRepository
	userRepo       repository.UserRepository
	merchantRepo   repository.MerchantRepository
	chapaClient    chapa.ChapaClient
	txManager      repository.TransactionManager
	enqueuer       jobs.Enqueuer
	feePercent     int64
	paymentTimeout time.Duration
	requirePhone   bool
}

type CheckoutConfig struct {
	FeePercent               int64
	PaymentTimeout           time.Duration
	RequirePhoneVerification bool
}

func NewCheckoutService(
	cartRepo repository.CartRepository,
	cartItemRepo repository.CartItemRepository,
	orderRepo repository.OrderRepository,
	paymentRepo repository.PaymentRepository,
	addressRepo repository.AddressRepository,
	inventoryRepo repository.InventoryRepository,
	productRepo repository.ProductRepository,
	variantRepo repository.ProductVariantRepository,
	userRepo repository.UserRepository,
	merchantRepo repository.MerchantRepository,
	chapaClient chapa.ChapaClient,
	txManager repository.TransactionManager,
	enqueuer jobs.Enqueuer,
	cfg CheckoutConfig,
) *CheckoutService {
	if cfg.PaymentTimeout <= 0 {
		cfg.PaymentTimeout = time.Hour
	}
	return &CheckoutService{
		cartRepo:       cartRepo,
		cartItemRepo:   cartItemRepo,
		orderRepo:      orderRepo,
		paymentRepo:    paymentRepo,
		addressRepo:    addressRepo,
		inventoryRepo:  inventoryRepo,
		productRepo:    productRepo,
		variantRepo:    variantRepo,
		userRepo:       userRepo,
		merchantRepo:   merchantRepo,
		chapaClient:    chapaClient,
		txManager:      txManager,
		enqueuer:       enqueuer,
		feePercent:     cfg.FeePercent,
		paymentTimeout: cfg.PaymentTimeout,
		requirePhone:   cfg.RequirePhoneVerification,
	}
}

type checkoutLine struct {
	item    domain.CartItem
	product *domain.Product
	variant *domain.ProductVariant
}

// ChapaTxRef builds a short, unique, order-traceable reference (about 45 chars).
func ChapaTxRef(orderID uuid.UUID, now time.Time) string {
	return "GB-" + strings.ReplaceAll(orderID.String(), "-", "") + "-" + strconv.FormatInt(now.Unix(), 36)
}

// Checkout turns the buyer's cart into a pending order and starts a Chapa payment.
// Stock stays reserved by the cart until Chapa accepts the payment; only then are
// the cart lines cleared so the reservation is owned by the order.
func (s *CheckoutService) Checkout(ctx context.Context, input CheckoutInput) (*CheckoutResult, error) {
	buyer, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if s.requirePhone && buyer.PhoneVerifiedAt == nil {
		return nil, domainerrors.Forbidden("verify your phone number before checking out")
	}

	cart, cartItems, err := s.loadCart(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	if err := s.checkAddress(ctx, input); err != nil {
		return nil, err
	}

	lines, merchantID, err := s.buildLines(ctx, cartItems)
	if err != nil {
		return nil, err
	}

	merchant, err := s.merchantRepo.GetByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if merchant.UserID == input.UserID {
		return nil, domainerrors.BadRequest("you cannot buy from your own shop")
	}

	pricing, err := s.price(lines, merchant, input.ShippingAddressID != nil)
	if err != nil {
		return nil, err
	}
	subtotalCents, deliveryFeeCents := pricing.subtotalCents, pricing.deliveryFeeCents
	platformFeeCents, totalCents := pricing.platformFeeCents, pricing.totalCents

	now := time.Now()
	orderID := uuid.New()
	chapaTxRef := ChapaTxRef(orderID, now)

	var order *domain.Order
	err = s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		txOrderRepo := s.orderRepo.WithTx(tx)

		order, err = txOrderRepo.Create(ctx, &domain.Order{
			ID:                orderID,
			BuyerID:           input.UserID,
			MerchantID:        merchantID,
			ShippingAddressID: input.ShippingAddressID,
			Status:            domain.OrderStatusPendingPayment,
			SubtotalCents:     subtotalCents,
			PlatformFeeCents:  platformFeeCents,
			DeliveryFeeCents:  deliveryFeeCents,
			TotalCents:        totalCents,
			Currency:          "ETB",
			Notes:             input.Notes,
			ChapaTxRef:        &chapaTxRef,
		})
		if err != nil {
			return err
		}

		for _, line := range lines {
			if _, err := txOrderRepo.CreateItem(ctx, &domain.OrderItem{
				ID:              uuid.New(),
				OrderID:         orderID,
				VariantID:       line.variant.ID,
				ProductName:     line.product.Name,
				VariantTitle:    line.variant.Title,
				Quantity:        line.item.Quantity,
				UnitPriceCents:  line.variant.PriceCents,
				TotalPriceCents: line.variant.PriceCents * int64(line.item.Quantity),
			}); err != nil {
				return err
			}
		}

		if _, err := s.paymentRepo.WithTx(tx).Create(ctx, &domain.Payment{
			ID:            uuid.New(),
			OrderID:       orderID,
			Provider:      "chapa",
			Status:        domain.PaymentStatusPending,
			AmountCents:   totalCents,
			Currency:      "ETB",
			ProviderTxRef: &chapaTxRef,
		}); err != nil {
			return err
		}

		return s.enqueuer.EnqueueTx(ctx, tx, jobs.ExpireUnpaidOrderArgs{OrderID: orderID}, &river.InsertOpts{
			ScheduledAt: now.Add(s.paymentTimeout),
		})
	})
	if err != nil {
		return nil, err
	}

	paymentReq := chapa.InitializePaymentRequest{
		Amount:   chapa.FormatAmount(totalCents),
		Currency: "ETB",
		TxRef:    chapaTxRef,
		Customization: &chapa.Customization{
			Title:       "Gebeya",
			Description: fmt.Sprintf("Order %s", orderID.String()[:8]),
		},
	}
	fillBuyerDetails(buyer, &paymentReq)

	chapaResp, err := s.chapaClient.InitializePayment(ctx, paymentReq)
	if err != nil {
		s.abandonOrder(ctx, order)
		return nil, domainerrors.Internal("failed to initialize payment", err)
	}

	if err := s.cartItemRepo.ClearCart(ctx, cart.ID); err != nil {
		slog.Error("CRITICAL: order created but cart lines not cleared; reservations are double-owned",
			"order_id", orderID, "cart_id", cart.ID, "error", err)
	}

	return &CheckoutResult{
		OrderID:    orderID,
		TotalCents: totalCents,
		Currency:   "ETB",
		PaymentURL: chapaResp.Data.CheckoutURL,
		ChapaTxRef: chapaTxRef,
	}, nil
}

type orderPricing struct {
	subtotalCents    int64
	deliveryFeeCents int64
	platformFeeCents int64
	totalCents       int64
}

func (s *CheckoutService) loadCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, []domain.CartItem, error) {
	cart, err := s.cartRepo.GetByUserID(ctx, userID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil, nil, domainerrors.BadRequest("cart is empty")
		}
		return nil, nil, err
	}

	items, err := s.cartItemRepo.ListByCart(ctx, cart.ID)
	if err != nil {
		return nil, nil, domainerrors.Internal("failed to get cart items", err)
	}
	if len(items) == 0 {
		return nil, nil, domainerrors.BadRequest("cart is empty")
	}
	return cart, items, nil
}

func (s *CheckoutService) checkAddress(ctx context.Context, input CheckoutInput) error {
	if input.ShippingAddressID == nil {
		return nil
	}
	address, err := s.addressRepo.GetByID(ctx, *input.ShippingAddressID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return domainerrors.BadRequest("shipping address not found")
		}
		return err
	}
	if address.UserID != input.UserID {
		return domainerrors.BadRequest("shipping address not found")
	}
	return nil
}

// price totals an order. Physical goods need an address and carry the merchant's
// delivery fee; the platform fee applies to the goods subtotal only.
func (s *CheckoutService) price(lines []checkoutLine, merchant *domain.MerchantProfile, hasAddress bool) (orderPricing, error) {
	var p orderPricing
	hasPhysical := false
	for _, line := range lines {
		p.subtotalCents += line.variant.PriceCents * int64(line.item.Quantity)
		if !line.product.IsDigital {
			hasPhysical = true
		}
	}

	if hasPhysical {
		if !hasAddress {
			return p, domainerrors.BadRequest("a delivery address is required for physical items")
		}
		p.deliveryFeeCents = merchant.DeliveryFeeFor(p.subtotalCents)
	}
	p.platformFeeCents = domain.CalculatePlatformFee(p.subtotalCents, s.feePercent)
	p.totalCents = p.subtotalCents + p.deliveryFeeCents
	return p, nil
}

func (s *CheckoutService) buildLines(ctx context.Context, items []domain.CartItem) ([]checkoutLine, uuid.UUID, error) {
	var merchantID uuid.UUID
	lines := make([]checkoutLine, 0, len(items))

	for i, item := range items {
		variant, err := s.variantRepo.GetByID(ctx, item.VariantID)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				return nil, uuid.Nil, domainerrors.BadRequest("a product in your cart is no longer available")
			}
			return nil, uuid.Nil, err
		}

		product, err := s.productRepo.GetByID(ctx, variant.ProductID)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				return nil, uuid.Nil, domainerrors.BadRequest("a product in your cart is no longer available")
			}
			return nil, uuid.Nil, err
		}

		if !variant.IsActive || product.Status != domain.ProductStatusActive {
			return nil, uuid.Nil, domainerrors.BadRequest(fmt.Sprintf("%s is no longer available", product.Name))
		}

		if i == 0 {
			merchantID = product.MerchantID
		} else if product.MerchantID != merchantID {
			return nil, uuid.Nil, domainerrors.BadRequest("all items in cart must be from the same merchant")
		}

		lines = append(lines, checkoutLine{item: item, product: product, variant: variant})
	}

	return lines, merchantID, nil
}

// fillBuyerDetails maps the buyer onto Chapa's fields. Chapa only accepts phone
// numbers in the 10-digit local form (09…/07…), so E.164 numbers are converted.
func fillBuyerDetails(user *domain.User, req *chapa.InitializePaymentRequest) {
	if user.Email != nil {
		req.Email = *user.Email
	}
	if user.Phone != nil {
		if local, err := phone.Local(*user.Phone); err == nil {
			req.Phone = local
		}
	}
	req.FirstName, req.LastName = "Gebeya", "Customer"
	if user.FullName != nil {
		first, last, _ := strings.Cut(strings.TrimSpace(*user.FullName), " ")
		if first != "" {
			req.FirstName = first
		}
		if last = strings.TrimSpace(last); last != "" {
			req.LastName = last
		}
	}
}

// abandonOrder cancels an order whose payment could not be started. Inventory is
// untouched because the cart still owns the reservations.
func (s *CheckoutService) abandonOrder(ctx context.Context, order *domain.Order) {
	err := s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := s.orderRepo.WithTx(tx).Transition(ctx, order.ID, domain.OrderStatusCancelled, domain.OrderStatusPendingPayment); err != nil {
			return err
		}
		payment, err := s.paymentRepo.WithTx(tx).GetByOrderID(ctx, order.ID)
		if err != nil {
			return err
		}
		_, err = s.paymentRepo.WithTx(tx).UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed)
		return err
	})
	if err != nil {
		slog.Error("failed to cancel order after payment initialization failure", "order_id", order.ID, "error", err)
	}
}
