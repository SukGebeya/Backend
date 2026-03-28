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
	"github.com/jackc/pgx/v5"
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
	blnkClient    blnk.BlnkClient
	chapaClient   chapa.ChapaClient
	txManager     repository.TransactionManager
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
	blnkClient blnk.BlnkClient,
	chapaClient chapa.ChapaClient,
	txManager repository.TransactionManager,
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
		txManager:     txManager,
	}
}

type checkoutTxResult struct {
	order      *domain.Order
	cartItems  []domain.CartItem
	cartID     uuid.UUID
	merchantID uuid.UUID
	totalCents int64
	chapaTxRef string
	escrowTxID string
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

	platformFeeCents := domain.CalculatePlatformFee(subtotalCents)
	totalCents := subtotalCents
	chapaTxRef := fmt.Sprintf("GEBEYA-%s-%d", uuid.New().String(), time.Now().Unix())

	var result *checkoutTxResult

	err = s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		txOrderRepo := s.orderRepo.WithTx(tx)
		txPaymentRepo := s.paymentRepo.WithTx(tx)
		txInventoryRepo := s.inventoryRepo.WithTx(tx)

		if err := s.reserveInventoryWithRepo(ctx, cartItems, txInventoryRepo); err != nil {
			return err
		}

		order, err := txOrderRepo.Create(ctx, &domain.Order{
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
			return domainerrors.Internal("failed to create order", err)
		}

		for _, item := range cartItems {
			variant, err := s.variantRepo.GetByID(ctx, item.VariantID)
			if err != nil {
				return domainerrors.Internal("failed to get variant", err)
			}

			_, err = txOrderRepo.CreateItem(ctx, &domain.OrderItem{
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
				return domainerrors.Internal("failed to create order item", err)
			}
		}

		_, err = txPaymentRepo.Create(ctx, &domain.Payment{
			ID:            uuid.New(),
			OrderID:       order.ID,
			Provider:      "chapa",
			Status:        domain.PaymentStatusPending,
			AmountCents:   totalCents,
			Currency:      "ETB",
			ProviderTxRef: &chapaTxRef,
		})
		if err != nil {
			return domainerrors.Internal("failed to create payment", err)
		}

		result = &checkoutTxResult{
			order:      order,
			cartItems:  cartItems,
			cartID:     cart.ID,
			merchantID: merchantID,
			totalCents: totalCents,
			chapaTxRef: chapaTxRef,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	escrowTxID, err := s.blnkClient.CreateEscrowHold(ctx, blnk.EscrowHoldRequest{
		OrderID:     result.order.ID,
		MerchantID:  result.merchantID,
		AmountCents: result.totalCents,
	})
	if err != nil {
		s.releaseInventory(ctx, result.cartItems)
		return nil, domainerrors.Internal("failed to create escrow hold", err)
	}

	if _, err := s.orderRepo.UpdateBlnkTransaction(ctx, result.order.ID, escrowTxID); err != nil {
		if voidErr := s.blnkClient.VoidEscrow(ctx, escrowTxID); voidErr != nil {
			slog.Error("CRITICAL: failed to void escrow after order update failure",
				"order_id", result.order.ID,
				"escrow_id", escrowTxID,
				"error", voidErr)
		}
		s.releaseInventory(ctx, result.cartItems)
		return nil, domainerrors.Internal("failed to associate escrow with order", err)
	}

	chapaResp, err := s.chapaClient.InitializePayment(ctx, chapa.InitializePaymentRequest{
		Amount:      fmt.Sprintf("%.2f", float64(result.totalCents)/100),
		Currency:    "ETB",
		TxRef:       result.chapaTxRef,
		CallbackURL: "",
		ReturnURL:   "",
		Description: fmt.Sprintf("Order %s", result.order.ID.String()),
	})
	if err != nil {
		if voidErr := s.blnkClient.VoidEscrow(ctx, escrowTxID); voidErr != nil {
			slog.Error("CRITICAL: failed to void escrow after payment init failure",
				"escrow_id", escrowTxID,
				"error", voidErr)
		}
		s.releaseInventory(ctx, result.cartItems)
		return nil, domainerrors.Internal("failed to initialize payment", err)
	}

	_ = s.cartItemRepo.ClearCart(ctx, result.cartID)

	return &CheckoutResult{
		OrderID:    result.order.ID,
		TotalCents: result.totalCents,
		Currency:   "ETB",
		PaymentURL: chapaResp.Data.CheckoutURL,
		ChapaTxRef: result.chapaTxRef,
	}, nil
}

func (s *CheckoutService) reserveInventoryWithRepo(ctx context.Context, items []domain.CartItem, inventoryRepo repository.InventoryRepository) error {
	for _, item := range items {
		if err := inventoryRepo.Reserve(ctx, item.VariantID, item.Quantity); err != nil {
			return domainerrors.Internal("failed to reserve inventory", err)
		}
	}
	return nil
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

func (s *CheckoutService) commitOrderInventory(ctx context.Context, orderID uuid.UUID) error {
	items, err := s.orderRepo.ListItemsByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order items: %w", err)
	}

	for _, item := range items {
		if err := s.inventoryRepo.Commit(ctx, item.VariantID, item.Quantity); err != nil {
			return fmt.Errorf("failed to commit inventory for variant %s: %w", item.VariantID, err)
		}
	}

	return nil
}

func (s *CheckoutService) ProcessSuccessfulPayment(ctx context.Context, chapaTxRef string) error {
	order, err := s.orderRepo.GetByChapaTxRef(ctx, chapaTxRef)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		return fmt.Errorf("failed to get payment: %w", err)
	}

	if _, err := s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusSucceeded); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	if _, err := s.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderStatusPaid); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	if err := s.commitOrderInventory(ctx, order.ID); err != nil {
		slog.Error("failed to commit inventory for order", "order_id", order.ID, "error", err)
		return fmt.Errorf("failed to commit inventory: %w", err)
	}

	return nil
}

func (s *CheckoutService) ProcessFailedPayment(ctx context.Context, chapaTxRef string) error {
	order, err := s.orderRepo.GetByChapaTxRef(ctx, chapaTxRef)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		return fmt.Errorf("failed to get payment: %w", err)
	}

	if _, err := s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	if _, err := s.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderStatusCancelled); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	if order.BlnkTransactionID != nil && *order.BlnkTransactionID != "" {
		if err := s.blnkClient.VoidEscrow(ctx, *order.BlnkTransactionID); err != nil {
			slog.Error("failed to void escrow on payment failure", "order_id", order.ID, "error", err)
			return fmt.Errorf("failed to void escrow: %w", err)
		}
	}

	if err := s.ReleaseOrderInventory(ctx, order.ID); err != nil {
		slog.Error("failed to release inventory on payment failure", "order_id", order.ID, "error", err)
		return fmt.Errorf("failed to release inventory: %w", err)
	}

	return nil
}
