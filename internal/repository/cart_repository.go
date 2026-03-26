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

type cartRepository struct {
	queries *db.Queries
}

func NewCartRepository(pool *pgxpool.Pool) CartRepository {
	return &cartRepository{queries: db.New(pool)}
}

var _ CartRepository = (*cartRepository)(nil)

func (r *cartRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Cart, error) {
	cart, err := r.queries.GetCartByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("cart not found")
		}
		return nil, domainerrors.Internal("failed to get cart", err)
	}
	return dbCartToDomain(cart), nil
}

func (r *cartRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	var pgUserID pgtype.UUID
	_ = pgUserID.Scan(userID.String())

	cart, err := r.queries.GetCartByUserID(ctx, pgUserID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("cart not found")
		}
		return nil, domainerrors.Internal("failed to get cart", err)
	}
	return dbCartToDomain(cart), nil
}

func (r *cartRepository) GetBySessionID(ctx context.Context, sessionID string) (*domain.Cart, error) {
	cart, err := r.queries.GetCartBySessionID(ctx, &sessionID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("cart not found")
		}
		return nil, domainerrors.Internal("failed to get cart", err)
	}
	return dbCartToDomain(cart), nil
}

func (r *cartRepository) Create(ctx context.Context, cart *domain.Cart) (*domain.Cart, error) {
	var userID pgtype.UUID
	if cart.UserID != nil {
		_ = userID.Scan(cart.UserID.String())
	}

	params := db.CreateCartParams{
		ID:        cart.ID,
		UserID:    userID,
		SessionID: cart.SessionID,
		Status:    db.CartStatus(cart.Status),
		ExpiresAt: pgtype.Timestamptz{Time: cart.ExpiresAt, Valid: true},
	}

	created, err := r.queries.CreateCart(ctx, params)
	if err != nil {
		return nil, domainerrors.Internal("failed to create cart", err)
	}
	return dbCartToDomain(created), nil
}

func (r *cartRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.CartStatus) (*domain.Cart, error) {
	cart, err := r.queries.UpdateCartStatus(ctx, db.UpdateCartStatusParams{
		ID:     id,
		Status: db.CartStatus(status),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("cart not found")
		}
		return nil, domainerrors.Internal("failed to update cart status", err)
	}
	return dbCartToDomain(cart), nil
}

func (r *cartRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteCart(ctx, id); err != nil {
		return domainerrors.Internal("failed to delete cart", err)
	}
	return nil
}

func (r *cartRepository) DeleteBySessionID(ctx context.Context, sessionID string) error {
	if err := r.queries.DeleteCartBySessionID(ctx, &sessionID); err != nil {
		return domainerrors.Internal("failed to delete cart", err)
	}
	return nil
}

func dbCartToDomain(c db.Cart) *domain.Cart {
	var userID *uuid.UUID
	if c.UserID.Valid {
		uid, _ := uuid.FromBytes(c.UserID.Bytes[:])
		userID = &uid
	}

	return &domain.Cart{
		ID:        c.ID,
		UserID:    userID,
		SessionID: c.SessionID,
		Status:    domain.CartStatus(c.Status),
		ExpiresAt: c.ExpiresAt.Time,
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
	}
}

type cartItemRepository struct {
	queries *db.Queries
}

func NewCartItemRepository(pool *pgxpool.Pool) CartItemRepository {
	return &cartItemRepository{queries: db.New(pool)}
}

var _ CartItemRepository = (*cartItemRepository)(nil)

func (r *cartItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.CartItem, error) {
	item, err := r.queries.GetCartItemByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("cart item not found")
		}
		return nil, domainerrors.Internal("failed to get cart item", err)
	}
	return dbCartItemToDomain(item), nil
}

func (r *cartItemRepository) GetByCartAndVariant(ctx context.Context, cartID, variantID uuid.UUID) (*domain.CartItem, error) {
	item, err := r.queries.GetCartItemByCartAndVariant(ctx, db.GetCartItemByCartAndVariantParams{
		CartID:    cartID,
		VariantID: variantID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("cart item not found")
		}
		return nil, domainerrors.Internal("failed to get cart item", err)
	}
	return dbCartItemToDomain(item), nil
}

func (r *cartItemRepository) ListByCart(ctx context.Context, cartID uuid.UUID) ([]domain.CartItem, error) {
	items, err := r.queries.ListCartItems(ctx, cartID)
	if err != nil {
		return nil, domainerrors.Internal("failed to list cart items", err)
	}
	result := make([]domain.CartItem, len(items))
	for i, item := range items {
		result[i] = *dbCartItemToDomain(item)
	}
	return result, nil
}

func (r *cartItemRepository) Create(ctx context.Context, item *domain.CartItem) (*domain.CartItem, error) {
	created, err := r.queries.CreateCartItem(ctx, db.CreateCartItemParams{
		ID:                 item.ID,
		CartID:             item.CartID,
		VariantID:          item.VariantID,
		Quantity:           int32(item.Quantity),
		PriceSnapshotCents: item.PriceSnapshotCents,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create cart item", err)
	}
	return dbCartItemToDomain(created), nil
}

func (r *cartItemRepository) Upsert(ctx context.Context, item *domain.CartItem) (*domain.CartItem, error) {
	created, err := r.queries.UpsertCartItem(ctx, db.UpsertCartItemParams{
		ID:                 item.ID,
		CartID:             item.CartID,
		VariantID:          item.VariantID,
		Quantity:           int32(item.Quantity),
		PriceSnapshotCents: item.PriceSnapshotCents,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to upsert cart item", err)
	}
	return dbCartItemToDomain(created), nil
}

func (r *cartItemRepository) UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) (*domain.CartItem, error) {
	item, err := r.queries.UpdateCartItemQuantity(ctx, db.UpdateCartItemQuantityParams{
		ID:       id,
		Quantity: int32(quantity),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("cart item not found")
		}
		return nil, domainerrors.Internal("failed to update cart item quantity", err)
	}
	return dbCartItemToDomain(item), nil
}

func (r *cartItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteCartItem(ctx, id); err != nil {
		return domainerrors.Internal("failed to delete cart item", err)
	}
	return nil
}

func (r *cartItemRepository) DeleteByCartAndVariant(ctx context.Context, cartID, variantID uuid.UUID) error {
	if err := r.queries.DeleteCartItemByCartAndVariant(ctx, db.DeleteCartItemByCartAndVariantParams{
		CartID:    cartID,
		VariantID: variantID,
	}); err != nil {
		return domainerrors.Internal("failed to delete cart item", err)
	}
	return nil
}

func (r *cartItemRepository) ClearCart(ctx context.Context, cartID uuid.UUID) error {
	if err := r.queries.ClearCartItems(ctx, cartID); err != nil {
		return domainerrors.Internal("failed to clear cart items", err)
	}
	return nil
}

func (r *cartItemRepository) CountByCart(ctx context.Context, cartID uuid.UUID) (int64, error) {
	count, err := r.queries.CountCartItems(ctx, cartID)
	if err != nil {
		return 0, domainerrors.Internal("failed to count cart items", err)
	}
	return count, nil
}

func (r *cartItemRepository) CountDistinctVariants(ctx context.Context, cartID uuid.UUID) (int, error) {
	count, err := r.queries.CountDistinctVariantsInCart(ctx, cartID)
	if err != nil {
		return 0, domainerrors.Internal("failed to count distinct variants", err)
	}
	return int(count), nil
}

func (r *cartItemRepository) MergeItems(ctx context.Context, sourceCartID, targetCartID uuid.UUID) error {
	newID := uuid.New()
	if err := r.queries.MergeCartItems(ctx, db.MergeCartItemsParams{
		NewID:        newID,
		TargetCartID: targetCartID,
		SourceCartID: sourceCartID,
	}); err != nil {
		return domainerrors.Internal("failed to merge cart items", err)
	}
	return nil
}

func dbCartItemToDomain(i db.CartItem) *domain.CartItem {
	return &domain.CartItem{
		ID:                 i.ID,
		CartID:             i.CartID,
		VariantID:          i.VariantID,
		Quantity:           int(i.Quantity),
		PriceSnapshotCents: i.PriceSnapshotCents,
		AddedAt:            i.AddedAt.Time,
	}
}
