package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

const (
	defaultJobTimeout = 30 * time.Second
)

var globalRiverClient *river.Client[pgx.Tx]

func SetRiverClient(client *river.Client[pgx.Tx]) {
	globalRiverClient = client
}

func EnqueueOrderNotification(ctx context.Context, orderID, buyerID uuid.UUID, event string, userRepo repository.UserRepository) {
	if globalRiverClient == nil {
		slog.Error("cannot enqueue notification: river client not initialized")
		return
	}

	user, err := userRepo.GetByID(ctx, buyerID)
	if err != nil {
		slog.Error("failed to get buyer for notification", "buyer_id", buyerID, "error", err)
		return
	}

	phone := ""
	if user.Phone != nil {
		phone = *user.Phone
	}

	if phone == "" {
		slog.Info("skipping notification: buyer has no phone", "order_id", orderID, "buyer_id", buyerID)
		return
	}

	_, err = globalRiverClient.Insert(ctx, SendOrderNotificationArgs{
		OrderID: orderID,
		Event:   event,
		Phone:   phone,
	}, nil)
	if err != nil {
		slog.Error("failed to enqueue notification job", "order_id", orderID, "event", event, "error", err)
		return
	}

	slog.Info("enqueued notification job", "order_id", orderID, "event", event)
}

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
	blnkClient    blnk.BlnkClient
	userRepo      repository.UserRepository
}

func NewProcessPaymentWebhookWorker(
	orderRepo repository.OrderRepository,
	paymentRepo repository.PaymentRepository,
	inventoryRepo repository.InventoryRepository,
	blnkClient blnk.BlnkClient,
	userRepo repository.UserRepository,
) *ProcessPaymentWebhookWorker {
	return &ProcessPaymentWebhookWorker{
		orderRepo:     orderRepo,
		paymentRepo:   paymentRepo,
		inventoryRepo: inventoryRepo,
		blnkClient:    blnkClient,
		userRepo:      userRepo,
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

		if err := w.commitOrderInventory(ctx, order.ID); err != nil {
			slog.Error("failed to commit inventory", "order_id", order.ID, "error", err)
			return err
		}

		EnqueueOrderNotification(ctx, order.ID, order.BuyerID, "payment_confirmed", w.userRepo)

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
				return err
			}
		}

		if err := w.releaseOrderInventory(ctx, order.ID); err != nil {
			slog.Error("failed to release order inventory", "order_id", order.ID, "error", err)
			return err
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

func (w *ProcessPaymentWebhookWorker) commitOrderInventory(ctx context.Context, orderID uuid.UUID) error {
	items, err := w.orderRepo.ListItemsByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to list order items: %w", err)
	}

	for _, item := range items {
		if err := w.inventoryRepo.Commit(ctx, item.VariantID, item.Quantity); err != nil {
			slog.Error("failed to commit inventory for order item",
				"order_id", orderID,
				"variant_id", item.VariantID,
				"quantity", item.Quantity,
				"error", err,
			)
			return fmt.Errorf("failed to commit inventory for variant %s: %w", item.VariantID, err)
		}
		slog.Info("committed inventory for order",
			"order_id", orderID,
			"variant_id", item.VariantID,
			"quantity", item.Quantity,
		)
	}

	return nil
}

func AddWorkers(workers *river.Workers, cartRepo repository.CartRepository, cartItemRepo repository.CartItemRepository, orderRepo repository.OrderRepository, paymentRepo repository.PaymentRepository, inventoryRepo repository.InventoryRepository, blnkClient blnk.BlnkClient, payoutRepo repository.PayoutRepository, merchantRepo repository.MerchantRepository, userRepo repository.UserRepository) {
	river.AddWorker(workers, NewReleaseCartReservationWorker(cartRepo, cartItemRepo, inventoryRepo))
	river.AddWorker(workers, NewMarkCartExpiredWorker(cartRepo, cartItemRepo))
	river.AddWorker(workers, NewProcessPaymentWebhookWorker(orderRepo, paymentRepo, inventoryRepo, blnkClient, userRepo))
	river.AddWorker(workers, NewSendOrderNotificationWorker())
	river.AddWorker(workers, NewProcessPayoutWorker(payoutRepo, merchantRepo, blnkClient))
}

type ProcessPayoutWorker struct {
	river.WorkerDefaults[ProcessPayoutArgs]
	payoutRepo   repository.PayoutRepository
	merchantRepo repository.MerchantRepository
	blnkClient   blnk.BlnkClient
}

func NewProcessPayoutWorker(payoutRepo repository.PayoutRepository, merchantRepo repository.MerchantRepository, blnkClient blnk.BlnkClient) *ProcessPayoutWorker {
	return &ProcessPayoutWorker{
		payoutRepo:   payoutRepo,
		merchantRepo: merchantRepo,
		blnkClient:   blnkClient,
	}
}

func (w *ProcessPayoutWorker) Timeout(job *river.Job[ProcessPayoutArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *ProcessPayoutWorker) Work(ctx context.Context, job *river.Job[ProcessPayoutArgs]) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	payoutID := job.Args.PayoutID

	slog.Info("processing payout", "payout_id", payoutID, "attempt", job.Attempt)

	payout, err := w.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			slog.Warn("payout not found", "payout_id", payoutID)
			return nil
		}
		slog.Error("failed to get payout", "payout_id", payoutID, "error", err)
		return err
	}

	if payout.Status != domain.PayoutStatusProcessing {
		slog.Info("payout is not in processing status, skipping", "payout_id", payoutID, "status", payout.Status)
		return nil
	}

	merchant, err := w.merchantRepo.GetByID(ctx, payout.MerchantID)
	if err != nil {
		slog.Error("failed to get merchant for payout", "payout_id", payoutID, "merchant_id", payout.MerchantID, "error", err)
		return err
	}

	if merchant.BlnkBalanceID == nil || *merchant.BlnkBalanceID == "" {
		slog.Error("merchant has no Blnk wallet", "payout_id", payoutID, "merchant_id", payout.MerchantID)
		failureReason := "Merchant has no Blnk wallet"
		_, _ = w.payoutRepo.UpdateStatus(ctx, payoutID, domain.PayoutStatusFailed, nil, &failureReason)
		return nil
	}

	blnkTxID, err := w.blnkClient.DebitMerchantForPayout(ctx, blnk.PayoutRequest{
		PayoutID:       payout.ID,
		MerchantID:     payout.MerchantID,
		AmountCents:    payout.AmountCents,
		Method:         string(payout.Method),
		Destination:    payout.Destination,
		MerchantBlnkID: *merchant.BlnkBalanceID,
	})
	if err != nil {
		slog.Error("failed to debit merchant wallet",
			"payout_id", payoutID,
			"merchant_id", payout.MerchantID,
			"error", err,
		)
		failureReason := "Failed to process payout with Blnk"
		_, _ = w.payoutRepo.UpdateStatus(ctx, payoutID, domain.PayoutStatusFailed, nil, &failureReason)
		return err
	}

	_, err = w.payoutRepo.UpdateStatus(ctx, payoutID, domain.PayoutStatusCompleted, &blnkTxID, nil)
	if err != nil {
		slog.Error("failed to update payout status to completed",
			"payout_id", payoutID,
			"blnk_tx_id", blnkTxID,
			"error", err,
		)
		return err
	}

	slog.Info("payout processed successfully",
		"payout_id", payoutID,
		"merchant_id", payout.MerchantID,
		"amount_cents", payout.AmountCents,
		"blnk_tx_id", blnkTxID,
	)

	return nil
}
