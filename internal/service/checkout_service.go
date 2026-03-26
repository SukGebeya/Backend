package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
)

var _ CheckoutManager = (*CheckoutService)(nil)

type CheckoutService struct {
	cartRepo      repository.CartRepository
	cartItemRepo  repository.CartItemRepository
	orderRepo     repository.OrderRepository
	paymentRepo   repository.PaymentRepository
	addressRepo   repository.AddressRepository
	inventoryRepo repository.InventoryRepository
	merchantRepo  repository.MerchantRepository
	productRepo   repository.ProductRepository
	variantRepo   repository.ProductVariantRepository
	blnkClient    *blnk.Client
	chapaClient   *chapa.Client
}

func NewCheckoutService(
	cartRepo repository.CartRepository,
	cartItemRepo repository.CartItemRepository,
	orderRepo repository.OrderRepository,
	paymentRepo repository.PaymentRepository,
	addressRepo repository.AddressRepository,
	inventoryRepo repository.InventoryRepository,
	merchantRepo repository.MerchantRepository,
	productRepo repository.ProductRepository,
	variantRepo repository.ProductVariantRepository,
	blnkClient *blnk.Client,
	chapaClient *chapa.Client,
) *CheckoutService {
	return &CheckoutService{
		cartRepo:      cartRepo,
		cartItemRepo:  cartItemRepo,
		orderRepo:     orderRepo,
		paymentRepo:   paymentRepo,
		addressRepo:   addressRepo,
		inventoryRepo: inventoryRepo,
		merchantRepo:  merchantRepo,
		productRepo:   productRepo,
		variantRepo:   variantRepo,
		blnkClient:    blnkClient,
		chapaClient:   chapaClient,
	}
}

func (s *CheckoutService) Checkout(ctx context.Context, input CheckoutInput) (*CheckoutResult, error) {
	cart, err := s.cartRepo.GetByUserID(ctx, input.UserID)
	if err != nil {
		return nil, domainerrors.NotFound("cart not found")
	}

	cartItems, err := s.cartItemRepo.ListByCart(ctx, cart.ID)
	if err != nil {
		return nil, domainerrors.Internal("failed to get cart items", err)
	}

	if len(cartItems) == 0 {
		return nil, domainerrors.BadRequest("cart is empty")
	}

	merchantID, subtotalCents, err := s.validateCartAndGetMerchant(ctx, cartItems)
	if err != nil {
		return nil, err
	}

	if err := s.reserveInventory(ctx, cartItems); err != nil {
		return nil, err
	}

	platformFeeCents := domain.CalculatePlatformFee(subtotalCents)
	totalCents := subtotalCents

	chapaTxRef := fmt.Sprintf("GEBEYA-%s-%d", uuid.New().String(), time.Now().Unix())

	order, err := s.orderRepo.Create(ctx, &domain.Order{
		ID:                uuid.New(),
		BuyerID:           input.UserID,
		MerchantID:        merchantID,
		ShippingAddressID: input.ShippingAddressID,
		Status:            domain.OrderStatusPendingPayment,
		SubtotalCents:     subtotalCents,
		PlatformFeeCents:  platformFeeCents,
		TotalCents:        totalCents,
		Currency:          "ETB",
		Notes:             input.Notes,
		ChapaTxRef:        &chapaTxRef,
	})
	if err != nil {
		s.releaseInventory(ctx, cartItems)
		return nil, domainerrors.Internal("failed to create order", err)
	}

	for _, item := range cartItems {
		variant, err := s.variantRepo.GetByID(ctx, item.VariantID)
		if err != nil {
			s.releaseInventory(ctx, cartItems)
			return nil, domainerrors.Internal("failed to get variant", err)
		}

		_, err = s.orderRepo.CreateItem(ctx, &domain.OrderItem{
			ID:              uuid.New(),
			OrderID:         order.ID,
			VariantID:       item.VariantID,
			ProductName:     variant.Title,
			VariantTitle:    variant.Title,
			Quantity:        item.Quantity,
			UnitPriceCents:  item.PriceSnapshotCents,
			TotalPriceCents: item.PriceSnapshotCents * int64(item.Quantity),
		})
		if err != nil {
			s.releaseInventory(ctx, cartItems)
			return nil, domainerrors.Internal("failed to create order item", err)
		}
	}

	payment, err := s.paymentRepo.Create(ctx, &domain.Payment{
		ID:            uuid.New(),
		OrderID:       order.ID,
		Provider:      "chapa",
		Status:        domain.PaymentStatusPending,
		AmountCents:   totalCents,
		Currency:      "ETB",
		ProviderTxRef: &chapaTxRef,
	})
	if err != nil {
		s.releaseInventory(ctx, cartItems)
		return nil, domainerrors.Internal("failed to create payment", err)
	}
	_ = payment

	escrowTxID, err := s.blnkClient.CreateEscrowHold(ctx, blnk.EscrowHoldRequest{
		OrderID:     order.ID,
		MerchantID:  merchantID,
		AmountCents: totalCents,
	})
	if err != nil {
		s.releaseInventory(ctx, cartItems)
		return nil, domainerrors.Internal("failed to create escrow hold", err)
	}

	if _, err := s.orderRepo.UpdateBlnkTransaction(ctx, order.ID, escrowTxID); err != nil {
		slog.Warn("failed to update order with escrow transaction", "order_id", order.ID, "error", err)
	}

	chapaResp, err := s.chapaClient.InitializePayment(ctx, chapa.InitializePaymentRequest{
		Amount:      fmt.Sprintf("%.2f", float64(totalCents)/100),
		Currency:    "ETB",
		TxRef:       chapaTxRef,
		CallbackURL: "",
		ReturnURL:   "",
		Description: fmt.Sprintf("Order %s", order.ID.String()),
	})
	if err != nil {
		s.releaseInventory(ctx, cartItems)
		return nil, domainerrors.Internal("failed to initialize payment", err)
	}

	_ = s.cartItemRepo.ClearCart(ctx, cart.ID)

	return &CheckoutResult{
		OrderID:    order.ID,
		TotalCents: totalCents,
		Currency:   "ETB",
		PaymentURL: chapaResp.Data.CheckoutURL,
		ChapaTxRef: chapaTxRef,
	}, nil
}

func (s *CheckoutService) validateCartAndGetMerchant(ctx context.Context, items []domain.CartItem) (uuid.UUID, int64, error) {
	var merchantID uuid.UUID
	var subtotalCents int64
	firstItem := true

	for _, item := range items {
		variant, err := s.variantRepo.GetByID(ctx, item.VariantID)
		if err != nil {
			return uuid.Nil, 0, domainerrors.Internal("failed to get variant", err)
		}

		product, err := s.productRepo.GetByID(ctx, variant.ProductID)
		if err != nil {
			return uuid.Nil, 0, domainerrors.Internal("failed to get product", err)
		}

		inventory, err := s.inventoryRepo.GetByVariantID(ctx, item.VariantID)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				return uuid.Nil, 0, domainerrors.BadRequest("variant inventory not found")
			}
			return uuid.Nil, 0, domainerrors.Internal("failed to check inventory", err)
		}

		availableQty := inventory.Quantity - inventory.Reserved
		if availableQty < item.Quantity {
			return uuid.Nil, 0, domainerrors.BadRequest(fmt.Sprintf("insufficient stock for variant %s: requested %d, available %d", variant.Title, item.Quantity, availableQty))
		}

		subtotalCents += item.PriceSnapshotCents * int64(item.Quantity)

		if firstItem {
			merchantID = product.MerchantID
			firstItem = false
		} else if product.MerchantID != merchantID {
			return uuid.Nil, 0, domainerrors.BadRequest("all items in cart must be from the same merchant")
		}
	}

	return merchantID, subtotalCents, nil
}

func (s *CheckoutService) reserveInventory(ctx context.Context, items []domain.CartItem) error {
	for _, item := range items {
		if err := s.inventoryRepo.Reserve(ctx, item.VariantID, item.Quantity); err != nil {
			s.releasePartialInventory(ctx, items, item.VariantID)
			return domainerrors.Internal("failed to reserve inventory", err)
		}
	}
	return nil
}

func (s *CheckoutService) releaseInventory(ctx context.Context, items []domain.CartItem) {
	for _, item := range items {
		if err := s.inventoryRepo.Release(ctx, item.VariantID, item.Quantity); err != nil {
			slog.Error("failed to release inventory", "variant_id", item.VariantID, "quantity", item.Quantity, "error", err)
		}
	}
}

func (s *CheckoutService) releasePartialInventory(ctx context.Context, items []domain.CartItem, upToVariantID uuid.UUID) {
	for _, item := range items {
		if item.VariantID == upToVariantID {
			break
		}
		if err := s.inventoryRepo.Release(ctx, item.VariantID, item.Quantity); err != nil {
			slog.Error("failed to release inventory", "variant_id", item.VariantID, "quantity", item.Quantity, "error", err)
		}
	}
}

func (s *CheckoutService) ReleaseOrderInventory(ctx context.Context, orderID uuid.UUID) error {
	items, err := s.orderRepo.ListItemsByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order items: %w", err)
	}

	for _, item := range items {
		if err := s.inventoryRepo.Release(ctx, item.VariantID, item.Quantity); err != nil {
			slog.Error("failed to release inventory for order item", "order_id", orderID, "variant_id", item.VariantID, "error", err)
		}
	}

	return nil
}

func (s *CheckoutService) ProcessSuccessfulPayment(ctx context.Context, chapaTxRef string) error {
	order, err := s.orderRepo.GetByChapaTxRef(ctx, chapaTxRef)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	if _, err := s.paymentRepo.UpdateStatus(ctx, order.ID, domain.PaymentStatusSucceeded); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	if _, err := s.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderStatusPaid); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

func (s *CheckoutService) ProcessFailedPayment(ctx context.Context, chapaTxRef string) error {
	order, err := s.orderRepo.GetByChapaTxRef(ctx, chapaTxRef)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	if _, err := s.paymentRepo.UpdateStatus(ctx, order.ID, domain.PaymentStatusFailed); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	if _, err := s.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderStatusCancelled); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	if order.BlnkTransactionID != nil && *order.BlnkTransactionID != "" {
		if err := s.blnkClient.VoidEscrow(ctx, *order.BlnkTransactionID); err != nil {
			slog.Warn("failed to void escrow on payment failure", "order_id", order.ID, "error", err)
		}
	}

	if err := s.ReleaseOrderInventory(ctx, order.ID); err != nil {
		slog.Warn("failed to release inventory on payment failure", "order_id", order.ID, "error", err)
	}

	return nil
}
