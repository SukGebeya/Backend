package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

const (
	defaultJobTimeout = 30 * time.Second
)

type ReleaseCartReservationWorker struct {
	river.WorkerDefaults[ReleaseCartReservationArgs]
	cartRepo      repository.CartRepository
	cartItemRepo  repository.CartItemRepository
	inventoryRepo repository.InventoryRepository
}

func NewReleaseCartReservationWorker(cartRepo repository.CartRepository, cartItemRepo repository.CartItemRepository, inventoryRepo repository.InventoryRepository) *ReleaseCartReservationWorker {
	return &ReleaseCartReservationWorker{
		cartRepo:      cartRepo,
		cartItemRepo:  cartItemRepo,
		inventoryRepo: inventoryRepo,
	}
}

func (w *ReleaseCartReservationWorker) Timeout(job *river.Job[ReleaseCartReservationArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *ReleaseCartReservationWorker) Work(ctx context.Context, job *river.Job[ReleaseCartReservationArgs]) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	cartID := job.Args.CartID

	slog.Info("releasing cart reservations", "cart_id", cartID, "attempt", job.Attempt)

	cart, err := w.cartRepo.GetByID(ctx, cartID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			slog.Info("cart no longer exists, skipping reservation release", "cart_id", cartID)
			return nil
		}
		slog.Error("failed to get cart for reservation release", "cart_id", cartID, "error", err)
		return err
	}

	if cart.Status != "active" {
		slog.Info("cart is no longer active, skipping reservation release", "cart_id", cartID, "status", cart.Status)
		return nil
	}

	cartItems, err := w.cartItemRepo.ListByCart(ctx, cartID)
	if err != nil {
		slog.Error("failed to get cart items for reservation release", "cart_id", cartID, "error", err)
		return err
	}

	for _, item := range cartItems {
		if err := w.inventoryRepo.Release(ctx, item.VariantID, item.Quantity); err != nil {
			slog.Error("failed to release inventory reservation",
				"cart_id", cartID,
				"variant_id", item.VariantID,
				"quantity", item.Quantity,
				"error", err,
			)
		} else {
			slog.Info("released inventory reservation",
				"cart_id", cartID,
				"variant_id", item.VariantID,
				"quantity", item.Quantity,
			)
		}
	}

	slog.Info("cart reservation release completed", "cart_id", cartID)

	return nil
}

type MarkCartExpiredWorker struct {
	river.WorkerDefaults[MarkCartExpiredArgs]
	cartRepo     repository.CartRepository
	cartItemRepo repository.CartItemRepository
}

func NewMarkCartExpiredWorker(cartRepo repository.CartRepository, cartItemRepo repository.CartItemRepository) *MarkCartExpiredWorker {
	return &MarkCartExpiredWorker{
		cartRepo:     cartRepo,
		cartItemRepo: cartItemRepo,
	}
}

func (w *MarkCartExpiredWorker) Timeout(job *river.Job[MarkCartExpiredArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *MarkCartExpiredWorker) Work(ctx context.Context, job *river.Job[MarkCartExpiredArgs]) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	cartID := job.Args.CartID

	slog.Info("processing cart expiry", "cart_id", cartID, "attempt", job.Attempt)

	cart, err := w.cartRepo.GetByID(ctx, cartID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			slog.Info("cart no longer exists, skipping expiry processing", "cart_id", cartID)
			return nil
		}
		slog.Error("failed to get cart for expiry check", "cart_id", cartID, "error", err)
		return err
	}

	if cart.Status != "active" {
		slog.Info("cart is no longer active, skipping expiry processing", "cart_id", cartID, "status", cart.Status)
		return nil
	}

	if _, err := w.cartRepo.UpdateStatus(ctx, cartID, domain.CartStatusAbandoned); err != nil {
		slog.Error("failed to mark cart as expired", "cart_id", cartID, "error", err)
		return err
	}

	slog.Info("cart marked as abandoned due to expiry", "cart_id", cartID)

	return nil
}

type ProcessPaymentWebhookWorker struct {
	river.WorkerDefaults[ProcessPaymentWebhookArgs]
	orderRepo     repository.OrderRepository
	paymentRepo   repository.PaymentRepository
	inventoryRepo repository.InventoryRepository
	blnkClient    *blnk.Client
}

func NewProcessPaymentWebhookWorker(orderRepo repository.OrderRepository, paymentRepo repository.PaymentRepository, inventoryRepo repository.InventoryRepository, blnkClient *blnk.Client) *ProcessPaymentWebhookWorker {
	return &ProcessPaymentWebhookWorker{
		orderRepo:     orderRepo,
		paymentRepo:   paymentRepo,
		inventoryRepo: inventoryRepo,
		blnkClient:    blnkClient,
	}
}

func (w *ProcessPaymentWebhookWorker) Timeout(job *river.Job[ProcessPaymentWebhookArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *ProcessPaymentWebhookWorker) Work(ctx context.Context, job *river.Job[ProcessPaymentWebhookArgs]) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	txRef := job.Args.TxRef
	status := job.Args.Status

	slog.Info("processing payment webhook", "tx_ref", txRef, "status", status, "attempt", job.Attempt)

	order, err := w.orderRepo.GetByChapaTxRef(ctx, txRef)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			slog.Warn("order not found for tx_ref", "tx_ref", txRef)
			return nil
		}
		slog.Error("failed to get order by tx_ref", "tx_ref", txRef, "error", err)
		return err
	}

	if order.Status != domain.OrderStatusPendingPayment {
		slog.Info("order already processed", "order_id", order.ID, "status", order.Status)
		return nil
	}

	payment, err := w.paymentRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		slog.Error("failed to get payment for order", "order_id", order.ID, "error", err)
		return err
	}

	if status == "success" {
		if _, err := w.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusSucceeded); err != nil {
			slog.Error("failed to update payment status", "payment_id", payment.ID, "error", err)
			return err
		}

		if _, err := w.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderStatusPaid); err != nil {
			slog.Error("failed to update order status", "order_id", order.ID, "error", err)
			return err
		}

		slog.Info("payment webhook processed successfully", "tx_ref", txRef, "order_id", order.ID)
	} else {
		if _, err := w.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed); err != nil {
			slog.Error("failed to update payment status to failed", "payment_id", payment.ID, "error", err)
			return err
		}

		if _, err := w.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderStatusCancelled); err != nil {
			slog.Error("failed to update order status to cancelled", "order_id", order.ID, "error", err)
			return err
		}

		if order.BlnkTransactionID != nil && *order.BlnkTransactionID != "" {
			if err := w.blnkClient.VoidEscrow(ctx, *order.BlnkTransactionID); err != nil {
				slog.Error("failed to void escrow", "order_id", order.ID, "blnk_txn_id", *order.BlnkTransactionID, "error", err)
			}
		}

		if err := w.releaseOrderInventory(ctx, order.ID); err != nil {
			slog.Error("failed to release order inventory", "order_id", order.ID, "error", err)
		}

		slog.Info("payment webhook processed as failed", "tx_ref", txRef, "order_id", order.ID)
	}

	return nil
}

func (w *ProcessPaymentWebhookWorker) releaseOrderInventory(ctx context.Context, orderID uuid.UUID) error {
	items, err := w.orderRepo.ListItemsByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := w.inventoryRepo.Release(ctx, item.VariantID, item.Quantity); err != nil {
			slog.Error("failed to release inventory for order item",
				"order_id", orderID,
				"variant_id", item.VariantID,
				"quantity", item.Quantity,
				"error", err,
			)
		} else {
			slog.Info("released inventory for failed order",
				"order_id", orderID,
				"variant_id", item.VariantID,
				"quantity", item.Quantity,
			)
		}
	}

	return nil
}

func AddWorkers(workers *river.Workers, cartRepo repository.CartRepository, cartItemRepo repository.CartItemRepository, orderRepo repository.OrderRepository, paymentRepo repository.PaymentRepository, inventoryRepo repository.InventoryRepository, blnkClient *blnk.Client) {
	river.AddWorker(workers, NewReleaseCartReservationWorker(cartRepo, cartItemRepo, inventoryRepo))
	river.AddWorker(workers, NewMarkCartExpiredWorker(cartRepo, cartItemRepo))
	river.AddWorker(workers, NewProcessPaymentWebhookWorker(orderRepo, paymentRepo, inventoryRepo, blnkClient))
}
