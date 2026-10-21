package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var _ jobs.PaymentProcessor = (*PaymentService)(nil)

type PaymentService struct {
	orderRepo       repository.OrderRepository
	paymentRepo     repository.PaymentRepository
	inventoryRepo   repository.InventoryRepository
	merchantRepo    repository.MerchantRepository
	walletRepo      repository.WalletTransactionRepository
	blnkClient      blnk.BlnkClient
	chapaClient     chapa.ChapaClient
	txManager       repository.TransactionManager
	enqueuer        jobs.Enqueuer
	requireLiveMode bool
}

type PaymentDeps struct {
	Orders          repository.OrderRepository
	Payments        repository.PaymentRepository
	Inventory       repository.InventoryRepository
	Merchants       repository.MerchantRepository
	WalletTxns      repository.WalletTransactionRepository
	Blnk            blnk.BlnkClient
	Chapa           chapa.ChapaClient
	TxManager       repository.TransactionManager
	Enqueuer        jobs.Enqueuer
	RequireLiveMode bool
}

func NewPaymentService(d PaymentDeps) *PaymentService {
	return &PaymentService{
		orderRepo:       d.Orders,
		paymentRepo:     d.Payments,
		inventoryRepo:   d.Inventory,
		merchantRepo:    d.Merchants,
		walletRepo:      d.WalletTxns,
		blnkClient:      d.Blnk,
		chapaClient:     d.Chapa,
		txManager:       d.TxManager,
		enqueuer:        d.Enqueuer,
		requireLiveMode: d.RequireLiveMode,
	}
}

type verifiedPayment struct {
	status      string
	amountCents int64
	chargeCents int64
	raw         []byte
}

// verify asks Chapa for the authoritative payment state. Webhook bodies are never
// trusted for money movement.
func (s *PaymentService) verify(ctx context.Context, txRef string) (*verifiedPayment, error) {
	resp, err := s.chapaClient.VerifyPayment(ctx, txRef)
	if err != nil {
		var apiErr *chapa.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return &verifiedPayment{status: chapa.StatusPending}, nil
		}
		return nil, err
	}

	raw, _ := json.Marshal(resp)
	result := &verifiedPayment{status: strings.ToLower(resp.Data.Status), raw: raw}
	if resp.Data.Amount != "" {
		cents, err := chapa.ParseAmountCents(resp.Data.Amount.String())
		if err != nil {
			return nil, fmt.Errorf("invalid amount from chapa verify: %w", err)
		}
		result.amountCents = cents
	}
	if resp.Data.Charge != "" {
		if cents, err := chapa.ParseAmountCents(resp.Data.Charge.String()); err == nil {
			result.chargeCents = cents
		}
	}
	if s.requireLiveMode && result.status == chapa.StatusSuccess && !strings.EqualFold(resp.Data.Mode, chapa.ModeLive) {
		slog.Error("CRITICAL: test-mode payment presented to production", "tx_ref", txRef, "mode", resp.Data.Mode)
		return nil, domainerrors.Conflict("payment was made in Chapa test mode")
	}
	return result, nil
}

// ProcessPayment reconciles an order with Chapa. It is safe to call repeatedly
// from webhooks, the return callback, polling, or the expiry job.
func (s *PaymentService) ProcessPayment(ctx context.Context, txRef string) error {
	order, err := s.orderRepo.GetByChapaTxRef(ctx, txRef)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			slog.Warn("payment for unknown tx_ref ignored", "tx_ref", txRef)
			return nil
		}
		return err
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		return err
	}

	if payment.Status == domain.PaymentStatusSucceeded || payment.Status == domain.PaymentStatusRefunded {
		return nil
	}

	verified, err := s.verify(ctx, txRef)
	if err != nil {
		return err
	}

	switch verified.status {
	case chapa.StatusSuccess:
		if verified.amountCents != order.TotalCents {
			slog.Error("CRITICAL: Chapa amount does not match order total",
				"order_id", order.ID, "tx_ref", txRef,
				"expected_cents", order.TotalCents, "paid_cents", verified.amountCents)
			return domainerrors.Conflict("paid amount does not match order total")
		}
		return s.markPaid(ctx, order, payment, verified)
	case chapa.StatusFailed, "cancelled":
		return s.markFailed(ctx, order, payment, verified.raw)
	default:
		return jobs.ErrNotReady
	}
}

func (s *PaymentService) markPaid(ctx context.Context, order *domain.Order, payment *domain.Payment, verified *verifiedPayment) error {
	if order.Status == domain.OrderStatusCancelled {
		return s.recordLatePayment(ctx, order, payment, verified)
	}
	if order.Status != domain.OrderStatusPendingPayment {
		return nil
	}

	escrowTxnID, err := s.blnkClient.CreateEscrowHold(ctx, blnk.EscrowHoldRequest{
		OrderID:     order.ID,
		MerchantID:  order.MerchantID,
		AmountCents: order.TotalCents,
	})
	if err != nil {
		return fmt.Errorf("failed to hold escrow: %w", err)
	}

	items, err := s.orderRepo.ListItemsByOrderID(ctx, order.ID)
	if err != nil {
		return err
	}

	return s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		txOrderRepo := s.orderRepo.WithTx(tx)

		if _, err := txOrderRepo.Transition(ctx, order.ID, domain.OrderStatusPaid, domain.OrderStatusPendingPayment); err != nil {
			return err
		}
		if _, err := txOrderRepo.UpdateBlnkTransaction(ctx, order.ID, escrowTxnID); err != nil {
			return err
		}
		if _, err := s.paymentRepo.WithTx(tx).UpdateResult(ctx, payment.ID, domain.PaymentStatusSucceeded, verified.raw, &escrowTxnID, &verified.chargeCents); err != nil {
			return err
		}

		txInventoryRepo := s.inventoryRepo.WithTx(tx)
		for _, item := range items {
			if err := txInventoryRepo.Commit(ctx, item.VariantID, item.Quantity); err != nil {
				slog.Error("inventory commit failed for paid order; stock needs review",
					"order_id", order.ID, "variant_id", item.VariantID, "quantity", item.Quantity, "error", err)
			}
		}

		slog.Info("payment confirmed", "order_id", order.ID, "escrow_txn_id", escrowTxnID)
		return s.enqueuer.EnqueueTx(ctx, tx, jobs.SendOrderNotificationArgs{OrderID: order.ID, Event: jobs.NotificationEventPaymentConfirmed}, nil)
	})
}

// recordLatePayment handles money that arrived after the order expired: the
// payment is recorded and refunded straight away.
func (s *PaymentService) recordLatePayment(ctx context.Context, order *domain.Order, payment *domain.Payment, verified *verifiedPayment) error {
	slog.Warn("payment received for cancelled order; refunding", "order_id", order.ID)
	return s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := s.paymentRepo.WithTx(tx).UpdateResult(ctx, payment.ID, domain.PaymentStatusSucceeded, verified.raw, nil, &verified.chargeCents); err != nil {
			return err
		}
		return s.enqueuer.EnqueueTx(ctx, tx, jobs.RefundOrderArgs{OrderID: order.ID}, nil)
	})
}

func (s *PaymentService) markFailed(ctx context.Context, order *domain.Order, payment *domain.Payment, raw []byte) error {
	if order.Status != domain.OrderStatusPendingPayment {
		return nil
	}

	items, err := s.orderRepo.ListItemsByOrderID(ctx, order.ID)
	if err != nil {
		return err
	}

	return s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := s.orderRepo.WithTx(tx).Transition(ctx, order.ID, domain.OrderStatusCancelled, domain.OrderStatusPendingPayment); err != nil {
			return err
		}
		if _, err := s.paymentRepo.WithTx(tx).UpdateResult(ctx, payment.ID, domain.PaymentStatusFailed, raw, nil, nil); err != nil {
			return err
		}
		releaseItems(ctx, s.inventoryRepo.WithTx(tx), order.ID, items)
		slog.Info("payment failed; order cancelled", "order_id", order.ID)
		return nil
	})
}

// ExpireUnpaidOrder runs when the payment window closes. Chapa is checked one last
// time so a payment whose webhook was lost is still honoured.
func (s *PaymentService) ExpireUnpaidOrder(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if order.Status != domain.OrderStatusPendingPayment || order.ChapaTxRef == nil {
		return nil
	}

	if err := s.ProcessPayment(ctx, *order.ChapaTxRef); err != nil && !errors.Is(err, jobs.ErrNotReady) {
		return err
	}

	order, err = s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != domain.OrderStatusPendingPayment {
		return nil
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	slog.Info("payment window expired; cancelling order", "order_id", orderID)
	return s.markFailed(ctx, order, payment, nil)
}

// RefundOrder returns a captured payment to the buyer: escrow is voided in Blnk and
// the money is refunded through Chapa. Each step is idempotent so the job can retry.
func (s *PaymentService) RefundOrder(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	if payment.Status == domain.PaymentStatusRefunded {
		return nil
	}
	if payment.Status != domain.PaymentStatusSucceeded {
		return domainerrors.BadRequest("payment was never captured; nothing to refund")
	}
	if order.Status != domain.OrderStatusCancelled {
		return domainerrors.BadRequest("only cancelled orders can be refunded")
	}

	if order.BlnkTransactionID != nil && *order.BlnkTransactionID != "" {
		if err := s.blnkClient.VoidInflight(ctx, *order.BlnkTransactionID); err != nil {
			return fmt.Errorf("failed to void escrow: %w", err)
		}
	}

	if order.ChapaTxRef != nil {
		if _, err := s.chapaClient.Refund(ctx, *order.ChapaTxRef, chapa.RefundRequest{
			Reason:    "Order cancelled",
			Reference: "refund-" + order.ID.String(),
		}); err != nil {
			return fmt.Errorf("chapa refund failed: %w", err)
		}
	}

	if err := s.bookProcessingFee(ctx, order.ID, payment.ChargeCents); err != nil {
		return err
	}

	return s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := s.paymentRepo.WithTx(tx).UpdateResult(ctx, payment.ID, domain.PaymentStatusRefunded, nil, nil, nil); err != nil {
			return err
		}
		if _, err := s.orderRepo.WithTx(tx).Transition(ctx, order.ID, domain.OrderStatusRefunded, domain.OrderStatusCancelled); err != nil {
			return err
		}
		slog.Info("order refunded", "order_id", order.ID, "amount_cents", payment.AmountCents)
		return s.enqueuer.EnqueueTx(ctx, tx, jobs.SendOrderNotificationArgs{OrderID: order.ID, Event: jobs.NotificationEventRefunded}, nil)
	})
}

// bookProcessingFee records Chapa's charge for an order against platform revenue.
// The Blnk reference is per order, so settlement, refunds and reversals can all
// call it and the fee is still booked exactly once.
func (s *PaymentService) bookProcessingFee(ctx context.Context, orderID uuid.UUID, chargeCents int64) error {
	return bookProcessingFee(ctx, s.blnkClient, orderID, chargeCents)
}

func bookProcessingFee(ctx context.Context, client blnk.BlnkClient, orderID uuid.UUID, chargeCents int64) error {
	if chargeCents <= 0 {
		return nil
	}
	if _, err := client.ChargeProcessingFee(ctx, blnk.ProcessingFeeRequest{OrderID: orderID, AmountCents: chargeCents}); err != nil {
		return fmt.Errorf("failed to book processing fee: %w", err)
	}
	return nil
}

// ProcessReversal handles money Chapa pulled back from a captured payment (a
// chargeback, or a refund issued from the Chapa dashboard). Before settlement the
// escrow is voided and the order cancelled; after settlement the merchant's wallet
// is debited, possibly going negative so future sales repay it.
func (s *PaymentService) ProcessReversal(ctx context.Context, txRef string) error {
	order, err := s.orderRepo.GetByChapaTxRef(ctx, txRef)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	payment, err := s.paymentRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		return err
	}
	if payment.Status != domain.PaymentStatusSucceeded {
		return nil
	}
	if order.Status == domain.OrderStatusCancelled || order.Status == domain.OrderStatusRefunded {
		return nil
	}

	if err := s.bookProcessingFee(ctx, order.ID, payment.ChargeCents); err != nil {
		return err
	}

	saleCredit, err := s.walletRepo.GetByOrderID(ctx, order.ID)
	if err == nil && saleCredit != nil {
		return s.reverseSettled(ctx, order, payment, saleCredit)
	}
	return s.reverseUnsettled(ctx, order, payment)
}

func (s *PaymentService) reverseUnsettled(ctx context.Context, order *domain.Order, payment *domain.Payment) error {
	if order.BlnkTransactionID != nil && *order.BlnkTransactionID != "" {
		if err := s.blnkClient.VoidInflight(ctx, *order.BlnkTransactionID); err != nil {
			return err
		}
	}
	items, err := s.orderRepo.ListItemsByOrderID(ctx, order.ID)
	if err != nil {
		return err
	}
	return s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := s.orderRepo.WithTx(tx).Transition(ctx, order.ID, domain.OrderStatusCancelled, order.Status); err != nil {
			return err
		}
		if _, err := s.paymentRepo.WithTx(tx).UpdateResult(ctx, payment.ID, domain.PaymentStatusReversed, nil, nil, nil); err != nil {
			return err
		}
		if order.Status == domain.OrderStatusPaid || order.Status == domain.OrderStatusProcessing {
			restockItems(ctx, s.inventoryRepo.WithTx(tx), order.ID, items)
		}
		slog.Warn("payment reversed before settlement; order cancelled", "order_id", order.ID)
		return s.enqueuer.EnqueueTx(ctx, tx, jobs.SendOrderNotificationArgs{OrderID: order.ID, Event: jobs.NotificationEventCancelled}, nil)
	})
}

func (s *PaymentService) reverseSettled(ctx context.Context, order *domain.Order, payment *domain.Payment, saleCredit *domain.WalletTransaction) error {
	merchant, err := s.merchantRepo.GetByID(ctx, order.MerchantID)
	if err != nil {
		return err
	}
	if merchant.BlnkBalanceID == nil {
		return domainerrors.BadRequest("merchant has no wallet")
	}

	reversalTxnID, err := s.blnkClient.DebitMerchantForReversal(ctx, blnk.ReversalRequest{
		OrderID:        order.ID,
		MerchantID:     order.MerchantID,
		AmountCents:    saleCredit.AmountCents,
		MerchantBlnkID: *merchant.BlnkBalanceID,
	})
	if err != nil {
		return err
	}

	return s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := s.paymentRepo.WithTx(tx).UpdateResult(ctx, payment.ID, domain.PaymentStatusReversed, nil, nil, nil); err != nil {
			return err
		}
		description := fmt.Sprintf("Payment reversal for order %s", order.ID)
		if _, err := s.walletRepo.WithTx(tx).Create(ctx, &domain.WalletTransaction{
			ID:                uuid.New(),
			MerchantID:        order.MerchantID,
			OrderID:           &order.ID,
			Type:              domain.WalletTransactionTypeReversalDebit,
			AmountCents:       saleCredit.AmountCents,
			Currency:          order.Currency,
			BlnkTransactionID: reversalTxnID,
			Status:            domain.WalletTransactionStatusApplied,
			Description:       &description,
		}); err != nil {
			return err
		}
		slog.Warn("payment reversed after settlement; merchant debited", "order_id", order.ID, "amount_cents", saleCredit.AmountCents)
		return nil
	})
}

func releaseItems(ctx context.Context, inventoryRepo repository.InventoryRepository, orderID uuid.UUID, items []domain.OrderItem) {
	for _, item := range items {
		if err := inventoryRepo.Release(ctx, item.VariantID, item.Quantity); err != nil {
			slog.Error("failed to release inventory", "order_id", orderID, "variant_id", item.VariantID, "quantity", item.Quantity, "error", err)
		}
	}
}

func restockItems(ctx context.Context, inventoryRepo repository.InventoryRepository, orderID uuid.UUID, items []domain.OrderItem) {
	for _, item := range items {
		if err := inventoryRepo.Restock(ctx, item.VariantID, item.Quantity); err != nil {
			slog.Error("failed to restock inventory", "order_id", orderID, "variant_id", item.VariantID, "quantity", item.Quantity, "error", err)
		}
	}
}
