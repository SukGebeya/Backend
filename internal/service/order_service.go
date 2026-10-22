package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

var _ OrderManager = (*OrderService)(nil)
var _ jobs.OrderSettler = (*OrderService)(nil)

const (
	CancelledByBuyer    = "buyer"
	CancelledByMerchant = "merchant"
)

type OrderService struct {
	orderRepo     repository.OrderRepository
	merchantRepo  repository.MerchantRepository
	shipmentRepo  repository.ShipmentRepository
	walletRepo    repository.WalletTransactionRepository
	paymentRepo   repository.PaymentRepository
	blnkClient    blnk.BlnkClient
	inventoryRepo repository.InventoryRepository
	txManager     repository.TransactionManager
	enqueuer      jobs.Enqueuer
	autoConfirm   time.Duration
	addressRepo   repository.AddressRepository
}

// WithAddresses lets order details include the delivery address, which the
// merchant needs to ship and cannot read through the buyer's address book.
func (s *OrderService) WithAddresses(repo repository.AddressRepository) *OrderService {
	s.addressRepo = repo
	return s
}

// WithAutoConfirm sets how long after shipping a delivery is confirmed on the
// buyer's behalf if they neither confirm nor dispute it.
func (s *OrderService) WithAutoConfirm(after time.Duration) *OrderService {
	s.autoConfirm = after
	return s
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	merchantRepo repository.MerchantRepository,
	shipmentRepo repository.ShipmentRepository,
	walletRepo repository.WalletTransactionRepository,
	paymentRepo repository.PaymentRepository,
	blnkClient blnk.BlnkClient,
	inventoryRepo repository.InventoryRepository,
	txManager repository.TransactionManager,
	enqueuer jobs.Enqueuer,
) *OrderService {
	return &OrderService{
		orderRepo:     orderRepo,
		merchantRepo:  merchantRepo,
		shipmentRepo:  shipmentRepo,
		walletRepo:    walletRepo,
		paymentRepo:   paymentRepo,
		blnkClient:    blnkClient,
		inventoryRepo: inventoryRepo,
		txManager:     txManager,
		enqueuer:      enqueuer,
		autoConfirm:   7 * 24 * time.Hour,
	}
}

func (s *OrderService) GetByID(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}

func (s *OrderService) GetByChapaTxRef(ctx context.Context, chapaTxRef string) (*domain.Order, error) {
	return s.orderRepo.GetByChapaTxRef(ctx, chapaTxRef)
}

func (s *OrderService) ListByBuyerID(ctx context.Context, buyerID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.Order], error) {
	orders, err := s.orderRepo.ListByBuyerID(ctx, buyerID, page, pageSize)
	if err != nil {
		return nil, err
	}

	total, err := s.orderRepo.CountByBuyerID(ctx, buyerID)
	if err != nil {
		return nil, err
	}

	return paginate(orders, total, page, pageSize), nil
}

func (s *OrderService) ListByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.Order], error) {
	orders, err := s.orderRepo.ListByMerchantID(ctx, merchantID, page, pageSize)
	if err != nil {
		return nil, err
	}

	total, err := s.orderRepo.CountByMerchantID(ctx, merchantID)
	if err != nil {
		return nil, err
	}

	return paginate(orders, total, page, pageSize), nil
}

func (s *OrderService) ListByStatus(ctx context.Context, status domain.OrderStatus, page, pageSize int) (*domain.PaginatedResult[domain.Order], error) {
	orders, err := s.orderRepo.ListByStatus(ctx, status, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &domain.PaginatedResult[domain.Order]{
		Items:   orders,
		Page:    page,
		PerPage: pageSize,
	}, nil
}

func paginate[T any](items []T, total int64, page, pageSize int) *domain.PaginatedResult[T] {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return &domain.PaginatedResult[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    pageSize,
		TotalPages: totalPages,
	}
}

func (s *OrderService) Create(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
	order := &domain.Order{
		ID:                uuid.New(),
		BuyerID:           input.BuyerID,
		MerchantID:        input.MerchantID,
		ShippingAddressID: input.ShippingAddressID,
		Status:            domain.OrderStatusPendingPayment,
		SubtotalCents:     input.SubtotalCents,
		PlatformFeeCents:  input.PlatformFeeCents,
		TotalCents:        input.TotalCents,
		Currency:          input.Currency,
		Notes:             input.Notes,
		ChapaTxRef:        &input.ChapaTxRef,
	}

	var created *domain.Order
	err := s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		txOrderRepo := s.orderRepo.WithTx(tx)
		var err error
		created, err = txOrderRepo.Create(ctx, order)
		if err != nil {
			return err
		}
		for _, item := range input.Items {
			if _, err := txOrderRepo.CreateItem(ctx, &domain.OrderItem{
				ID:              uuid.New(),
				OrderID:         created.ID,
				VariantID:       item.VariantID,
				ProductName:     item.ProductName,
				VariantTitle:    item.VariantTitle,
				Quantity:        item.Quantity,
				UnitPriceCents:  item.UnitPriceCents,
				TotalPriceCents: item.TotalPriceCents,
			}); err != nil {
				return fmt.Errorf("failed to create order item: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// UpdateStatus is the merchant-facing status endpoint. Only side-effect-free
// transitions are allowed here; shipping, delivery, cancellation and payment
// each have dedicated flows that move money and stock.
func (s *OrderService) UpdateStatus(ctx context.Context, orderID uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	switch status {
	case domain.OrderStatusProcessing:
		return s.startProcessing(ctx, orderID)
	case domain.OrderStatusShipped:
		return s.MarkOrderShipped(ctx, MarkOrderShippedInput{OrderID: orderID})
	case domain.OrderStatusCancelled:
		return s.CancelOrder(ctx, orderID, CancelledByMerchant)
	default:
		return nil, domainerrors.BadRequest(fmt.Sprintf("status %s cannot be set directly", status))
	}
}

func (s *OrderService) startProcessing(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	var updated *domain.Order
	err := s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		updated, err = s.orderRepo.WithTx(tx).Transition(ctx, orderID, domain.OrderStatusProcessing, domain.OrderStatusPaid)
		if err != nil {
			return err
		}
		_, err = s.shipmentRepo.WithTx(tx).Create(ctx, &domain.Shipment{
			ID:      uuid.New(),
			OrderID: orderID,
			Status:  domain.ShipmentStatusPreparing,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *OrderService) UpdateBlnkTransaction(ctx context.Context, orderID uuid.UUID, blnkTxID string) (*domain.Order, error) {
	return s.orderRepo.UpdateBlnkTransaction(ctx, orderID, blnkTxID)
}

func (s *OrderService) GetOrderWithItems(ctx context.Context, orderID uuid.UUID) (*domain.OrderWithItems, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	items, err := s.orderRepo.ListItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	return &domain.OrderWithItems{
		Order: *order,
		Items: items,
	}, nil
}

func (s *OrderService) GetOrderWithDetails(ctx context.Context, orderID uuid.UUID) (*OrderWithDetails, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	items, err := s.orderRepo.ListItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	var shipment *domain.Shipment
	if order.Status == domain.OrderStatusProcessing ||
		order.Status == domain.OrderStatusShipped ||
		order.Status == domain.OrderStatusDelivered {
		shipment, _ = s.shipmentRepo.GetByOrderID(ctx, orderID)
	}

	// The address is a reference, not a snapshot: a buyer who deletes it leaves the order without one.
	var address *domain.Address
	if s.addressRepo != nil && order.ShippingAddressID != nil {
		address, err = s.addressRepo.GetByID(ctx, *order.ShippingAddressID)
		if err != nil && !domainerrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get shipping address: %w", err)
		}
	}

	return &OrderWithDetails{
		Order:           *order,
		Items:           items,
		Shipment:        shipment,
		ShippingAddress: address,
	}, nil
}

// ConfirmDelivery records the buyer's confirmation and schedules settlement in
// the same transaction, so escrow release cannot be lost or applied twice.
func (s *OrderService) ConfirmDelivery(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	var updated *domain.Order
	err := s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		updated, err = s.orderRepo.WithTx(tx).Transition(ctx, orderID, domain.OrderStatusDelivered, domain.OrderStatusShipped)
		if err != nil {
			if appErr, ok := domainerrors.IsAppError(err); ok && appErr.Code == domainerrors.CodeConflict {
				return domainerrors.BadRequest("order must be shipped before confirming delivery")
			}
			return err
		}

		txShipmentRepo := s.shipmentRepo.WithTx(tx)
		if shipment, err := txShipmentRepo.GetByOrderID(ctx, orderID); err == nil {
			if _, err := txShipmentRepo.UpdateDelivered(ctx, shipment.ID); err != nil {
				return err
			}
		}

		if err := s.enqueuer.EnqueueTx(ctx, tx, jobs.SettleOrderArgs{OrderID: orderID}, nil); err != nil {
			return err
		}
		return s.enqueuer.EnqueueTx(ctx, tx, jobs.SendOrderNotificationArgs{OrderID: orderID, Event: jobs.NotificationEventDelivered}, nil)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// SettleOrder releases escrow for a delivered order: the inflight hold is committed,
// the merchant is credited the net amount and the platform keeps its fee. Every
// Blnk call is keyed by an order-derived reference, so retries never double-pay.
func (s *OrderService) SettleOrder(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != domain.OrderStatusDelivered {
		return domainerrors.BadRequest("only delivered orders can be settled")
	}
	if order.BlnkTransactionID == nil || *order.BlnkTransactionID == "" {
		return domainerrors.BadRequest("order has no escrow transaction")
	}

	if existing, err := s.walletRepo.GetByOrderID(ctx, orderID); err == nil && existing != nil {
		return nil
	}

	merchant, err := s.merchantRepo.GetByID(ctx, order.MerchantID)
	if err != nil {
		return err
	}
	if merchant.BlnkBalanceID == nil || *merchant.BlnkBalanceID == "" {
		return domainerrors.BadRequest("merchant has no wallet")
	}

	if err := s.blnkClient.CommitInflight(ctx, *order.BlnkTransactionID); err != nil {
		return err
	}

	merchantTxnID, err := s.blnkClient.CreditMerchant(ctx, blnk.MerchantCreditRequest{
		OrderID:          order.ID,
		MerchantID:       order.MerchantID,
		SubtotalCents:    order.SubtotalCents,
		FeeCents:         order.PlatformFeeCents,
		DeliveryFeeCents: order.DeliveryFeeCents,
		MerchantBlnkID:   *merchant.BlnkBalanceID,
	})
	if err != nil {
		return err
	}

	if order.PlatformFeeCents > 0 {
		if _, err := s.blnkClient.CreditPlatformFee(ctx, blnk.PlatformFeeRequest{
			OrderID:  order.ID,
			FeeCents: order.PlatformFeeCents,
		}); err != nil {
			return err
		}
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		return err
	}
	if err := bookProcessingFee(ctx, s.blnkClient, order.ID, payment.ChargeCents); err != nil {
		return err
	}

	netAmountCents := domain.CalculateMerchantPayout(order.SubtotalCents, order.PlatformFeeCents, order.DeliveryFeeCents)
	description := fmt.Sprintf("Sale credit for order %s", order.ID)
	if err := s.merchantRepo.IncrementTotalSales(ctx, order.MerchantID, order.SubtotalCents); err != nil {
		return err
	}
	if _, err := s.walletRepo.Create(ctx, &domain.WalletTransaction{
		ID:                uuid.New(),
		MerchantID:        order.MerchantID,
		OrderID:           &order.ID,
		Type:              domain.WalletTransactionTypeSaleCredit,
		AmountCents:       netAmountCents,
		Currency:          order.Currency,
		BlnkTransactionID: merchantTxnID,
		Status:            domain.WalletTransactionStatusApplied,
		Description:       &description,
	}); err != nil {
		return err
	}

	slog.Info("order settled", "order_id", order.ID, "merchant_id", order.MerchantID, "net_cents", netAmountCents, "fee_cents", order.PlatformFeeCents)
	return nil
}

// AutoConfirmDelivery runs when the buyer neither confirmed nor disputed a shipped
// order in time; the delivery is confirmed so the merchant gets paid.
func (s *OrderService) AutoConfirmDelivery(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if order.Status != domain.OrderStatusShipped {
		return nil
	}
	if _, err := s.ConfirmDelivery(ctx, orderID); err != nil {
		return err
	}
	slog.Info("delivery auto-confirmed", "order_id", orderID)
	return nil
}

// OpenDispute freezes an order the buyer has a problem with (never delivered,
// wrong or damaged goods). Escrow stays held until an admin resolves it.
func (s *OrderService) OpenDispute(ctx context.Context, orderID uuid.UUID, reason string) (*domain.Order, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, domainerrors.BadRequest("a reason is required to open a dispute")
	}
	order, err := s.orderRepo.OpenDispute(ctx, orderID, reason, domain.OrderStatusProcessing, domain.OrderStatusShipped)
	if err != nil {
		return nil, err
	}
	slog.Warn("order dispute opened", "order_id", orderID)
	return order, nil
}

type DisputeOutcome string

const (
	DisputeRelease DisputeOutcome = "release"
	DisputeRefund  DisputeOutcome = "refund"
)

// ResolveDispute settles a disputed order: release pays the merchant as if the
// delivery were confirmed; refund cancels the order and returns the buyer's money.
func (s *OrderService) ResolveDispute(ctx context.Context, orderID uuid.UUID, outcome DisputeOutcome, restock bool) (*domain.Order, error) {
	var updated *domain.Order
	err := s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		switch outcome {
		case DisputeRelease:
			updated, err = s.orderRepo.WithTx(tx).Transition(ctx, orderID, domain.OrderStatusDelivered, domain.OrderStatusDisputed)
			if err != nil {
				return err
			}
			txShipmentRepo := s.shipmentRepo.WithTx(tx)
			if shipment, err := txShipmentRepo.GetByOrderID(ctx, orderID); err == nil {
				if _, err := txShipmentRepo.UpdateDelivered(ctx, shipment.ID); err != nil {
					return err
				}
			}
			return s.enqueuer.EnqueueTx(ctx, tx, jobs.SettleOrderArgs{OrderID: orderID}, nil)
		case DisputeRefund:
			updated, err = s.orderRepo.WithTx(tx).Transition(ctx, orderID, domain.OrderStatusCancelled, domain.OrderStatusDisputed)
			if err != nil {
				return err
			}
			if restock {
				items, err := s.orderRepo.WithTx(tx).ListItemsByOrderID(ctx, orderID)
				if err != nil {
					return err
				}
				restockItems(ctx, s.inventoryRepo.WithTx(tx), orderID, items)
			}
			return s.enqueuer.EnqueueTx(ctx, tx, jobs.RefundOrderArgs{OrderID: orderID}, nil)
		default:
			return domainerrors.BadRequest("outcome must be release or refund")
		}
	})
	if err != nil {
		return nil, err
	}
	slog.Info("order dispute resolved", "order_id", orderID, "outcome", outcome)
	return updated, nil
}

func (s *OrderService) MerchantOrderCounts(ctx context.Context, merchantID uuid.UUID) (map[domain.OrderStatus]int64, error) {
	return s.orderRepo.CountByStatusForMerchant(ctx, merchantID)
}

func (s *OrderService) MarkOrderShipped(ctx context.Context, input MarkOrderShippedInput) (*domain.Order, error) {
	var updated *domain.Order
	err := s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		updated, err = s.orderRepo.WithTx(tx).Transition(ctx, input.OrderID, domain.OrderStatusShipped, domain.OrderStatusProcessing)
		if err != nil {
			if appErr, ok := domainerrors.IsAppError(err); ok && appErr.Code == domainerrors.CodeConflict {
				return domainerrors.BadRequest("order must be in processing status to ship")
			}
			return err
		}

		txShipmentRepo := s.shipmentRepo.WithTx(tx)
		shipment, err := txShipmentRepo.GetByOrderID(ctx, input.OrderID)
		if err != nil {
			if !domainerrors.IsNotFound(err) {
				return err
			}
			shipment, err = txShipmentRepo.Create(ctx, &domain.Shipment{
				ID:      uuid.New(),
				OrderID: input.OrderID,
				Status:  domain.ShipmentStatusPreparing,
			})
			if err != nil {
				return err
			}
		}

		if _, err := txShipmentRepo.UpdateShipped(ctx, shipment.ID, input.Carrier, input.TrackingNumber, input.EstimatedDelivery); err != nil {
			return err
		}

		if err := s.enqueuer.EnqueueTx(ctx, tx, jobs.AutoConfirmDeliveryArgs{OrderID: input.OrderID}, &river.InsertOpts{
			ScheduledAt: time.Now().Add(s.autoConfirm),
		}); err != nil {
			return err
		}

		return s.enqueuer.EnqueueTx(ctx, tx, jobs.SendOrderNotificationArgs{OrderID: input.OrderID, Event: jobs.NotificationEventShipped}, nil)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// CancelOrder cancels an order on behalf of the buyer or merchant. Unpaid orders
// release their stock reservation; paid orders are restocked and refunded.
func (s *OrderService) CancelOrder(ctx context.Context, orderID uuid.UUID, cancelledBy string) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	allowed := map[string][]domain.OrderStatus{
		CancelledByBuyer:    {domain.OrderStatusPendingPayment, domain.OrderStatusPaid},
		CancelledByMerchant: {domain.OrderStatusPendingPayment, domain.OrderStatusPaid, domain.OrderStatusProcessing},
	}[cancelledBy]

	cancellable := false
	for _, st := range allowed {
		if order.Status == st {
			cancellable = true
			break
		}
	}
	if !cancellable {
		return nil, domainerrors.BadRequest(fmt.Sprintf("cannot cancel order in %s status", order.Status))
	}

	items, err := s.orderRepo.ListItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	var updated *domain.Order
	err = s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		updated, err = s.orderRepo.WithTx(tx).Transition(ctx, orderID, domain.OrderStatusCancelled, order.Status)
		if err != nil {
			return err
		}

		txInventoryRepo := s.inventoryRepo.WithTx(tx)
		if order.Status == domain.OrderStatusPendingPayment {
			releaseItems(ctx, txInventoryRepo, orderID, items)
			payment, err := s.paymentRepo.WithTx(tx).GetByOrderID(ctx, orderID)
			if err == nil {
				if _, err := s.paymentRepo.WithTx(tx).UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed); err != nil {
					return err
				}
			}
		} else {
			restockItems(ctx, txInventoryRepo, orderID, items)
			if err := s.enqueuer.EnqueueTx(ctx, tx, jobs.RefundOrderArgs{OrderID: orderID}, nil); err != nil {
				return err
			}
		}

		return s.enqueuer.EnqueueTx(ctx, tx, jobs.SendOrderNotificationArgs{OrderID: orderID, Event: jobs.NotificationEventCancelled}, nil)
	})
	if err != nil {
		return nil, err
	}

	slog.Info("order cancelled", "order_id", orderID, "by", cancelledBy, "previous_status", order.Status)
	return updated, nil
}
