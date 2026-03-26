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
	blnkClient   *blnk.Client
}

func NewOrderService(orderRepo repository.OrderRepository, merchantRepo repository.MerchantRepository, blnkClient *blnk.Client) *OrderService {
	return &OrderService{
		orderRepo:    orderRepo,
		merchantRepo: merchantRepo,
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
	return s.orderRepo.UpdateStatus(ctx, orderID, status)
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

	if _, err := s.blnkClient.CreditMerchant(ctx, merchantCreditReq); err != nil {
		return nil, fmt.Errorf("failed to credit merchant: %w", err)
	}

	updated, err := s.orderRepo.UpdateStatus(ctx, orderID, domain.OrderStatusDelivered)
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	return updated, nil
}
