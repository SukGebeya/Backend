package service

import (
	"context"
	"fmt"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
)

var _ OrderManager = (*OrderService)(nil)

type OrderService struct {
	orderRepo    repository.OrderRepository
	merchantRepo repository.MerchantRepository
	shipmentRepo repository.ShipmentRepository
	walletRepo   repository.WalletTransactionRepository
	blnkClient   *blnk.Client
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	merchantRepo repository.MerchantRepository,
	shipmentRepo repository.ShipmentRepository,
	walletRepo repository.WalletTransactionRepository,
	blnkClient *blnk.Client,
) *OrderService {
	return &OrderService{
		orderRepo:    orderRepo,
		merchantRepo: merchantRepo,
		shipmentRepo: shipmentRepo,
		walletRepo:   walletRepo,
		blnkClient:   blnkClient,
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

	return &domain.PaginatedResult[domain.Order]{
		Items:      orders,
		Total:      total,
		Page:       page,
		PerPage:    pageSize,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
	}, nil
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

	return &domain.PaginatedResult[domain.Order]{
		Items:      orders,
		Total:      total,
		Page:       page,
		PerPage:    pageSize,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
	}, nil
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

	created, err := s.orderRepo.Create(ctx, order)
	if err != nil {
		return nil, err
	}

	for _, item := range input.Items {
		orderItem := &domain.OrderItem{
			ID:              uuid.New(),
			OrderID:         created.ID,
			VariantID:       item.VariantID,
			ProductName:     item.ProductName,
			VariantTitle:    item.VariantTitle,
			Quantity:        item.Quantity,
			UnitPriceCents:  item.UnitPriceCents,
			TotalPriceCents: item.TotalPriceCents,
		}
		if _, err := s.orderRepo.CreateItem(ctx, orderItem); err != nil {
			return nil, fmt.Errorf("failed to create order item: %w", err)
		}
	}

	return created, nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, orderID uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if !s.isValidStatusTransition(order.Status, status) {
		return nil, domainerrors.BadRequest(fmt.Sprintf("cannot transition from %s to %s", order.Status, status))
	}

	updated, err := s.orderRepo.UpdateStatus(ctx, orderID, status)
	if err != nil {
		return nil, err
	}

	if status == domain.OrderStatusProcessing {
		if _, err := s.shipmentRepo.Create(ctx, &domain.Shipment{
			ID:      uuid.New(),
			OrderID: orderID,
			Status:  domain.ShipmentStatusPreparing,
		}); err != nil {
			return nil, fmt.Errorf("failed to create shipment: %w", err)
		}
	}

	return updated, nil
}

func (s *OrderService) isValidStatusTransition(from, to domain.OrderStatus) bool {
	validTransitions := map[domain.OrderStatus][]domain.OrderStatus{
		domain.OrderStatusPendingPayment: {domain.OrderStatusPaid, domain.OrderStatusCancelled},
		domain.OrderStatusPaid:           {domain.OrderStatusProcessing, domain.OrderStatusCancelled},
		domain.OrderStatusProcessing:     {domain.OrderStatusShipped, domain.OrderStatusCancelled},
		domain.OrderStatusShipped:        {domain.OrderStatusDelivered, domain.OrderStatusCancelled},
		domain.OrderStatusDelivered:      {},
		domain.OrderStatusCancelled:      {domain.OrderStatusRefunded},
		domain.OrderStatusRefunded:       {},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
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

	return &OrderWithDetails{
		Order:    *order,
		Items:    items,
		Shipment: shipment,
	}, nil
}

func (s *OrderService) ConfirmDelivery(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status != domain.OrderStatusShipped {
		return nil, domainerrors.BadRequest("order must be shipped before confirming delivery")
	}

	if order.BlnkTransactionID == nil || *order.BlnkTransactionID == "" {
		return nil, domainerrors.BadRequest("order has no escrow transaction")
	}

	merchant, err := s.merchantRepo.GetByID(ctx, order.MerchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	if merchant.BlnkBalanceID == nil || *merchant.BlnkBalanceID == "" {
		return nil, domainerrors.BadRequest("merchant has no Blnk wallet")
	}

	if err := s.blnkClient.CommitEscrow(ctx, *order.BlnkTransactionID); err != nil {
		return nil, fmt.Errorf("failed to commit escrow: %w", err)
	}

	merchantCreditReq := blnk.MerchantCreditRequest{
		OrderID:        order.ID,
		MerchantID:     order.MerchantID,
		SubtotalCents:  order.SubtotalCents,
		FeeCents:       order.PlatformFeeCents,
		MerchantBlnkID: *merchant.BlnkBalanceID,
	}

	blnkTxnID, err := s.blnkClient.CreditMerchant(ctx, merchantCreditReq)
	if err != nil {
		return nil, fmt.Errorf("failed to credit merchant: %w", err)
	}

	netAmountCents := order.SubtotalCents - order.PlatformFeeCents
	description := fmt.Sprintf("Sale credit for order %s", order.ID.String())

	walletTxn := &domain.WalletTransaction{
		ID:                uuid.New(),
		MerchantID:        order.MerchantID,
		OrderID:           &order.ID,
		Type:              domain.WalletTransactionTypeSaleCredit,
		AmountCents:       netAmountCents,
		Currency:          order.Currency,
		BlnkTransactionID: blnkTxnID,
		Status:            domain.WalletTransactionStatusApplied,
		Description:       &description,
	}

	if _, err := s.walletRepo.Create(ctx, walletTxn); err != nil {
		return nil, fmt.Errorf("failed to create wallet transaction: %w", err)
	}

	shipment, err := s.shipmentRepo.GetByOrderID(ctx, orderID)
	if err == nil && shipment != nil {
		if _, err := s.shipmentRepo.UpdateDelivered(ctx, shipment.ID); err != nil {
			return nil, fmt.Errorf("failed to update shipment status: %w", err)
		}
	}

	updated, err := s.orderRepo.UpdateStatus(ctx, orderID, domain.OrderStatusDelivered)
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	return updated, nil
}

func (s *OrderService) MarkOrderShipped(ctx context.Context, input MarkOrderShippedInput) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, input.OrderID)
	if err != nil {
		return nil, err
	}

	if order.Status != domain.OrderStatusProcessing {
		return nil, domainerrors.BadRequest("order must be in processing status to ship")
	}

	shipment, err := s.shipmentRepo.GetByOrderID(ctx, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}

	if _, err := s.shipmentRepo.UpdateShipped(ctx, shipment.ID, input.Carrier, input.TrackingNumber, input.EstimatedDelivery); err != nil {
		return nil, fmt.Errorf("failed to update shipment: %w", err)
	}

	return s.orderRepo.UpdateStatus(ctx, input.OrderID, domain.OrderStatusShipped)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID uuid.UUID, cancelledBy string) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	cancellableStatuses := map[domain.OrderStatus]bool{
		domain.OrderStatusPendingPayment: true,
		domain.OrderStatusPaid:           true,
		domain.OrderStatusProcessing:     true,
	}

	if !cancellableStatuses[order.Status] {
		return nil, domainerrors.BadRequest(fmt.Sprintf("cannot cancel order in %s status", order.Status))
	}

	if order.BlnkTransactionID != nil && *order.BlnkTransactionID != "" {
		if err := s.blnkClient.VoidEscrow(ctx, *order.BlnkTransactionID); err != nil {
			return nil, fmt.Errorf("failed to void escrow: %w", err)
		}
	}

	updated, err := s.orderRepo.UpdateStatus(ctx, orderID, domain.OrderStatusCancelled)
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	return updated, nil
}
