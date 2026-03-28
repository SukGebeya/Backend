package repository

import (
	"context"

	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type orderRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &orderRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

func (r *orderRepository) WithTx(tx pgx.Tx) OrderRepository {
	return &orderRepository{
		pool:    r.pool,
		queries: r.queries.WithTx(tx),
	}
}

var _ OrderRepository = (*orderRepository)(nil)

func (r *orderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	order, err := r.queries.GetOrderByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("order not found")
		}
		return nil, domainerrors.Internal("failed to get order", err)
	}
	return dbOrderToDomain(order), nil
}

func (r *orderRepository) GetByChapaTxRef(ctx context.Context, chapaTxRef string) (*domain.Order, error) {
	order, err := r.queries.GetOrderByChapaTxRef(ctx, &chapaTxRef)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("order not found")
		}
		return nil, domainerrors.Internal("failed to get order", err)
	}
	return dbOrderToDomain(order), nil
}

func (r *orderRepository) ListByBuyerID(ctx context.Context, buyerID uuid.UUID, page, pageSize int) ([]domain.Order, error) {
	offset := (page - 1) * pageSize
	orders, err := r.queries.ListOrdersByBuyerID(ctx, db.ListOrdersByBuyerIDParams{
		BuyerID:    buyerID,
		PageSize:   int32(pageSize),
		PageOffset: int32(offset),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list orders", err)
	}
	result := make([]domain.Order, len(orders))
	for i, o := range orders {
		result[i] = *dbOrderToDomain(o)
	}
	return result, nil
}

func (r *orderRepository) ListByMerchantID(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]domain.Order, error) {
	offset := (page - 1) * pageSize
	orders, err := r.queries.ListOrdersByMerchantID(ctx, db.ListOrdersByMerchantIDParams{
		MerchantID: merchantID,
		PageSize:   int32(pageSize),
		PageOffset: int32(offset),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list orders", err)
	}
	result := make([]domain.Order, len(orders))
	for i, o := range orders {
		result[i] = *dbOrderToDomain(o)
	}
	return result, nil
}

func (r *orderRepository) ListByStatus(ctx context.Context, status domain.OrderStatus, page, pageSize int) ([]domain.Order, error) {
	offset := (page - 1) * pageSize
	orders, err := r.queries.ListOrdersByStatus(ctx, db.ListOrdersByStatusParams{
		Status:     db.OrderStatus(status),
		PageSize:   int32(pageSize),
		PageOffset: int32(offset),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list orders", err)
	}
	result := make([]domain.Order, len(orders))
	for i, o := range orders {
		result[i] = *dbOrderToDomain(o)
	}
	return result, nil
}

func (r *orderRepository) Create(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	var shippingAddressID pgtype.UUID
	if order.ShippingAddressID != nil {
		shippingAddressID = pgtype.UUID{Bytes: *order.ShippingAddressID, Valid: true}
	}

	var notes *string
	if order.Notes != nil {
		notes = order.Notes
	}

	var chapaTxRef *string
	if order.ChapaTxRef != nil {
		chapaTxRef = order.ChapaTxRef
	}

	created, err := r.queries.CreateOrder(ctx, db.CreateOrderParams{
		ID:                order.ID,
		BuyerID:           order.BuyerID,
		MerchantID:        order.MerchantID,
		ShippingAddressID: shippingAddressID,
		Status:            db.OrderStatus(order.Status),
		SubtotalCents:     order.SubtotalCents,
		PlatformFeeCents:  order.PlatformFeeCents,
		TotalCents:        order.TotalCents,
		Currency:          order.Currency,
		Notes:             notes,
		ChapaTxRef:        chapaTxRef,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create order", err)
	}
	return dbOrderToDomain(created), nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	order, err := r.queries.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		ID:     id,
		Status: db.OrderStatus(status),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("order not found")
		}
		return nil, domainerrors.Internal("failed to update order status", err)
	}
	return dbOrderToDomain(order), nil
}

func (r *orderRepository) UpdateBlnkTransaction(ctx context.Context, id uuid.UUID, blnkTxID string) (*domain.Order, error) {
	order, err := r.queries.UpdateOrderBlnkTransaction(ctx, db.UpdateOrderBlnkTransactionParams{
		ID:                id,
		BlnkTransactionID: &blnkTxID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("order not found")
		}
		return nil, domainerrors.Internal("failed to update order", err)
	}
	return dbOrderToDomain(order), nil
}

func (r *orderRepository) CountByBuyerID(ctx context.Context, buyerID uuid.UUID) (int64, error) {
	count, err := r.queries.CountOrdersByBuyerID(ctx, buyerID)
	if err != nil {
		return 0, domainerrors.Internal("failed to count orders", err)
	}
	return count, nil
}

func (r *orderRepository) CountByMerchantID(ctx context.Context, merchantID uuid.UUID) (int64, error) {
	count, err := r.queries.CountOrdersByMerchantID(ctx, merchantID)
	if err != nil {
		return 0, domainerrors.Internal("failed to count orders", err)
	}
	return count, nil
}

func (r *orderRepository) GetItemByID(ctx context.Context, id uuid.UUID) (*domain.OrderItem, error) {
	item, err := r.queries.GetOrderItemByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("order item not found")
		}
		return nil, domainerrors.Internal("failed to get order item", err)
	}
	return dbOrderItemToDomain(item), nil
}

func (r *orderRepository) ListItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	items, err := r.queries.ListOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, domainerrors.Internal("failed to list order items", err)
	}
	result := make([]domain.OrderItem, len(items))
	for i, item := range items {
		result[i] = *dbOrderItemToDomain(item)
	}
	return result, nil
}

func (r *orderRepository) CreateItem(ctx context.Context, item *domain.OrderItem) (*domain.OrderItem, error) {
	created, err := r.queries.CreateOrderItem(ctx, db.CreateOrderItemParams{
		ID:              item.ID,
		OrderID:         item.OrderID,
		VariantID:       item.VariantID,
		ProductName:     item.ProductName,
		VariantTitle:    item.VariantTitle,
		Quantity:        int32(item.Quantity),
		UnitPriceCents:  item.UnitPriceCents,
		TotalPriceCents: item.TotalPriceCents,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create order item", err)
	}
	return dbOrderItemToDomain(created), nil
}

func dbOrderToDomain(o db.Order) *domain.Order {
	var shippingAddressID *uuid.UUID
	if o.ShippingAddressID.Valid {
		id := uuid.UUID(o.ShippingAddressID.Bytes)
		shippingAddressID = &id
	}

	return &domain.Order{
		ID:                o.ID,
		BuyerID:           o.BuyerID,
		MerchantID:        o.MerchantID,
		ShippingAddressID: shippingAddressID,
		Status:            domain.OrderStatus(o.Status),
		SubtotalCents:     o.SubtotalCents,
		PlatformFeeCents:  o.PlatformFeeCents,
		TotalCents:        o.TotalCents,
		Currency:          o.Currency,
		Notes:             o.Notes,
		BlnkTransactionID: o.BlnkTransactionID,
		ChapaTxRef:        o.ChapaTxRef,
		CreatedAt:         o.CreatedAt.Time,
		UpdatedAt:         o.UpdatedAt.Time,
	}
}

func dbOrderItemToDomain(i db.OrderItem) *domain.OrderItem {
	return &domain.OrderItem{
		ID:              i.ID,
		OrderID:         i.OrderID,
		VariantID:       i.VariantID,
		ProductName:     i.ProductName,
		VariantTitle:    i.VariantTitle,
		Quantity:        int(i.Quantity),
		UnitPriceCents:  i.UnitPriceCents,
		TotalPriceCents: i.TotalPriceCents,
	}
}
