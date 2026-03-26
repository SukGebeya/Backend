package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
)

var _ CartManager = (*CartService)(nil)

type CartService struct {
	cartRepo      repository.CartRepository
	cartItemRepo  repository.CartItemRepository
	sessionRepo   repository.SessionRepository
	variantRepo   repository.ProductVariantRepository
	inventoryRepo repository.InventoryRepository
	jobScheduler  JobScheduler
}

func NewCartService(
	cartRepo repository.CartRepository,
	cartItemRepo repository.CartItemRepository,
	sessionRepo repository.SessionRepository,
	variantRepo repository.ProductVariantRepository,
	inventoryRepo repository.InventoryRepository,
	jobScheduler JobScheduler,
) *CartService {
	return &CartService{
		cartRepo:      cartRepo,
		cartItemRepo:  cartItemRepo,
		sessionRepo:   sessionRepo,
		variantRepo:   variantRepo,
		inventoryRepo: inventoryRepo,
		jobScheduler:  jobScheduler,
	}
}

func (s *CartService) GetOrCreateCart(ctx context.Context, userID *uuid.UUID, sessionID *string) (*domain.Cart, error) {
	if userID != nil {
		return s.getOrCreateUserCart(ctx, *userID)
	}
	if sessionID != nil && *sessionID != "" {
		return s.getOrCreateSessionCart(ctx, *sessionID)
	}
	return nil, domainerrors.BadRequest("either user_id or session_id is required")
}

func (s *CartService) getOrCreateUserCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	cart, err := s.cartRepo.GetByUserID(ctx, userID)
	if err == nil {
		return s.checkExpiry(ctx, cart)
	}

	if !domainerrors.IsNotFound(err) {
		return nil, err
	}

	cart = &domain.Cart{
		ID:        uuid.New(),
		UserID:    &userID,
		Status:    domain.CartStatusActive,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	created, err := s.cartRepo.Create(ctx, cart)
	if err != nil {
		return nil, domainerrors.Internal("failed to create cart", err)
	}

	s.scheduleCartExpiry(ctx, created.ID, created.ExpiresAt)

	return created, nil
}

func (s *CartService) getOrCreateSessionCart(ctx context.Context, sessionID string) (*domain.Cart, error) {
	valid, err := s.sessionRepo.Get(ctx, sessionID)
	if err != nil {
		return nil, domainerrors.Internal("failed to validate session", err)
	}
	if !valid {
		if err := s.sessionRepo.Create(ctx, sessionID, int64(7*24*60*60)); err != nil {
			return nil, domainerrors.Internal("failed to create session", err)
		}
	}

	cart, err := s.cartRepo.GetBySessionID(ctx, sessionID)
	if err == nil {
		return s.checkExpiry(ctx, cart)
	}

	if !domainerrors.IsNotFound(err) {
		return nil, err
	}

	cart = &domain.Cart{
		ID:        uuid.New(),
		SessionID: &sessionID,
		Status:    domain.CartStatusActive,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	created, err := s.cartRepo.Create(ctx, cart)
	if err != nil {
		return nil, domainerrors.Internal("failed to create cart", err)
	}

	s.scheduleCartExpiry(ctx, created.ID, created.ExpiresAt)

	return created, nil
}

func (s *CartService) scheduleCartExpiry(ctx context.Context, cartID uuid.UUID, expiresAt time.Time) {
	if s.jobScheduler == nil {
		return
	}

	if err := s.jobScheduler.ScheduleCartExpiry(ctx, cartID, expiresAt); err != nil {
		slog.Error("failed to schedule cart expiry job", "cart_id", cartID, "error", err)
	}
}

func (s *CartService) checkExpiry(ctx context.Context, cart *domain.Cart) (*domain.Cart, error) {
	if time.Now().After(cart.ExpiresAt) {
		_, err := s.cartRepo.UpdateStatus(ctx, cart.ID, domain.CartStatusAbandoned)
		if err != nil {
			slog.Error("failed to update expired cart status", "cart_id", cart.ID, "error", err)
		}
		return nil, domainerrors.NotFound("cart has expired")
	}
	return cart, nil
}

func (s *CartService) GetCart(ctx context.Context, cartID uuid.UUID) (*domain.Cart, error) {
	cart, err := s.cartRepo.GetByID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	return s.checkExpiry(ctx, cart)
}

func (s *CartService) AddItem(ctx context.Context, cartID uuid.UUID, variantID uuid.UUID, quantity int, priceCents int64) (*domain.CartItem, error) {
	if quantity <= 0 {
		return nil, domainerrors.BadRequest("quantity must be greater than 0")
	}

	count, err := s.cartItemRepo.CountDistinctVariants(ctx, cartID)
	if err != nil {
		return nil, domainerrors.Internal("failed to count cart items", err)
	}

	existing, err := s.cartItemRepo.GetByCartAndVariant(ctx, cartID, variantID)
	if err == nil {
		newQty := existing.Quantity + quantity
		if newQty > 100 {
			return nil, domainerrors.BadRequest("maximum quantity per item is 100")
		}

		inventory, err := s.inventoryRepo.GetByVariantID(ctx, variantID)
		if err != nil {
			if !domainerrors.IsNotFound(err) {
				return nil, domainerrors.Internal("failed to check inventory", err)
			}
		} else {
			available := inventory.Quantity - inventory.Reserved
			if available < quantity {
				return nil, domainerrors.BadRequest("insufficient inventory available")
			}
			if err := s.inventoryRepo.Reserve(ctx, variantID, quantity); err != nil {
				return nil, domainerrors.Internal("failed to reserve inventory", err)
			}
		}

		updated, err := s.cartItemRepo.UpdateQuantity(ctx, existing.ID, newQty)
		if err != nil {
			s.inventoryRepo.Release(ctx, variantID, quantity)
			return nil, domainerrors.Internal("failed to update cart item", err)
		}
		return updated, nil
	}

	if !domainerrors.IsNotFound(err) {
		return nil, domainerrors.Internal("failed to check existing item", err)
	}

	if count >= domain.MaxCartItems {
		return nil, domainerrors.BadRequest("cart cannot contain more than 50 different items")
	}

	inventory, err := s.inventoryRepo.GetByVariantID(ctx, variantID)
	if err != nil {
		if !domainerrors.IsNotFound(err) {
			return nil, domainerrors.Internal("failed to check inventory", err)
		}
	} else {
		available := inventory.Quantity - inventory.Reserved
		if available < quantity {
			return nil, domainerrors.BadRequest("insufficient inventory available")
		}
		if err := s.inventoryRepo.Reserve(ctx, variantID, quantity); err != nil {
			return nil, domainerrors.Internal("failed to reserve inventory", err)
		}
	}

	item := &domain.CartItem{
		ID:                 uuid.New(),
		CartID:             cartID,
		VariantID:          variantID,
		Quantity:           quantity,
		PriceSnapshotCents: priceCents,
	}

	created, err := s.cartItemRepo.Create(ctx, item)
	if err != nil {
		if inventory != nil {
			s.inventoryRepo.Release(ctx, variantID, quantity)
		}
		return nil, domainerrors.Internal("failed to add item to cart", err)
	}

	return created, nil
}

func (s *CartService) UpdateItemQuantity(ctx context.Context, cartID, itemID uuid.UUID, quantity int) (*domain.CartItem, error) {
	if quantity <= 0 {
		if err := s.cartItemRepo.Delete(ctx, itemID); err != nil {
			return nil, domainerrors.Internal("failed to remove item", err)
		}
		return nil, nil
	}

	if quantity > 100 {
		return nil, domainerrors.BadRequest("maximum quantity per item is 100")
	}

	item, err := s.cartItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}

	if item.CartID != cartID {
		return nil, domainerrors.Forbidden("item does not belong to this cart")
	}

	return s.cartItemRepo.UpdateQuantity(ctx, itemID, quantity)
}

func (s *CartService) RemoveItem(ctx context.Context, cartID, itemID uuid.UUID) error {
	item, err := s.cartItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return err
	}

	if item.CartID != cartID {
		return domainerrors.Forbidden("item does not belong to this cart")
	}

	if err := s.cartItemRepo.Delete(ctx, itemID); err != nil {
		return domainerrors.Internal("failed to remove item from cart", err)
	}

	if err := s.inventoryRepo.Release(ctx, item.VariantID, item.Quantity); err != nil {
		slog.Error("failed to release inventory reservation", "variant_id", item.VariantID, "quantity", item.Quantity, "error", err)
	}

	return nil
}

func (s *CartService) RemoveItemByVariant(ctx context.Context, cartID, variantID uuid.UUID) error {
	item, err := s.cartItemRepo.GetByCartAndVariant(ctx, cartID, variantID)
	if err != nil {
		return err
	}

	if err := s.cartItemRepo.Delete(ctx, item.ID); err != nil {
		return domainerrors.Internal("failed to remove item from cart", err)
	}

	if err := s.inventoryRepo.Release(ctx, variantID, item.Quantity); err != nil {
		slog.Error("failed to release inventory reservation", "variant_id", variantID, "quantity", item.Quantity, "error", err)
	}

	return nil
}

func (s *CartService) UpdateItemQuantityByVariant(ctx context.Context, cartID, variantID uuid.UUID, quantity int) (*domain.CartItem, error) {
	if quantity <= 0 {
		if err := s.RemoveItemByVariant(ctx, cartID, variantID); err != nil {
			return nil, err
		}
		return nil, nil
	}

	if quantity > 100 {
		return nil, domainerrors.BadRequest("maximum quantity per item is 100")
	}

	item, err := s.cartItemRepo.GetByCartAndVariant(ctx, cartID, variantID)
	if err != nil {
		return nil, err
	}

	if quantity == item.Quantity {
		return item, nil
	}

	inventory, err := s.inventoryRepo.GetByVariantID(ctx, variantID)
	if err != nil {
		if !domainerrors.IsNotFound(err) {
			return nil, domainerrors.Internal("failed to check inventory", err)
		}
	} else {
		if quantity > item.Quantity {
			additionalQty := quantity - item.Quantity
			available := inventory.Quantity - inventory.Reserved
			if available < additionalQty {
				return nil, domainerrors.BadRequest("insufficient inventory available")
			}
			if err := s.inventoryRepo.Reserve(ctx, variantID, additionalQty); err != nil {
				return nil, domainerrors.Internal("failed to reserve inventory", err)
			}
		} else {
			releaseQty := item.Quantity - quantity
			if err := s.inventoryRepo.Release(ctx, variantID, releaseQty); err != nil {
				slog.Error("failed to release inventory reservation", "variant_id", variantID, "quantity", releaseQty, "error", err)
			}
		}
	}

	updated, err := s.cartItemRepo.UpdateQuantity(ctx, item.ID, quantity)
	if err != nil {
		if inventory != nil && quantity > item.Quantity {
			releaseQty := quantity - item.Quantity
			s.inventoryRepo.Release(ctx, variantID, releaseQty)
		}
		return nil, domainerrors.Internal("failed to update cart item quantity", err)
	}

	return updated, nil
}

func (s *CartService) ClearCart(ctx context.Context, cartID uuid.UUID) error {
	items, err := s.cartItemRepo.ListByCart(ctx, cartID)
	if err != nil {
		return domainerrors.Internal("failed to list cart items", err)
	}

	for _, item := range items {
		if err := s.inventoryRepo.Release(ctx, item.VariantID, item.Quantity); err != nil {
			slog.Error("failed to release inventory reservation", "variant_id", item.VariantID, "quantity", item.Quantity, "error", err)
		}
	}

	return s.cartItemRepo.ClearCart(ctx, cartID)
}

func (s *CartService) MergeCart(ctx context.Context, userID uuid.UUID, sessionID string) (*domain.Cart, error) {
	userCart, err := s.getOrCreateUserCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	sessionCart, err := s.cartRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return userCart, nil
		}
		return nil, domainerrors.Internal("failed to get session cart", err)
	}

	if err := s.cartItemRepo.MergeItems(ctx, sessionCart.ID, userCart.ID); err != nil {
		return nil, domainerrors.Internal("failed to merge cart items", err)
	}

	if err := s.cartRepo.DeleteBySessionID(ctx, sessionID); err != nil {
		slog.Error("failed to delete session cart after merge", "session_id", sessionID, "error", err)
	}

	if err := s.sessionRepo.Delete(ctx, sessionID); err != nil {
		slog.Error("failed to delete session after merge", "session_id", sessionID, "error", err)
	}

	items, err := s.cartItemRepo.ListByCart(ctx, userCart.ID)
	if err != nil {
		return nil, domainerrors.Internal("failed to get cart items", err)
	}

	userCart.Items = items
	return userCart, nil
}

func (s *CartService) GetCartWithItems(ctx context.Context, cartID uuid.UUID) (*domain.CartWithDetails, error) {
	cart, err := s.GetCart(ctx, cartID)
	if err != nil {
		return nil, err
	}

	items, err := s.cartItemRepo.ListByCart(ctx, cartID)
	if err != nil {
		return nil, domainerrors.Internal("failed to get cart items", err)
	}

	result := &domain.CartWithDetails{
		Cart:  *cart,
		Items: make([]domain.CartItemWithDetails, len(items)),
	}

	for i, item := range items {
		variant, err := s.variantRepo.GetByID(ctx, item.VariantID)
		if err != nil {
			slog.Error("failed to get variant for cart item", "variant_id", item.VariantID, "error", err)
			result.Items[i] = domain.CartItemWithDetails{
				CartItem:     item,
				VariantTitle: "Unknown",
				Currency:     "ETB",
			}
			continue
		}

		result.Items[i] = domain.CartItemWithDetails{
			CartItem:     item,
			VariantTitle: variant.Title,
			Currency:     variant.Currency,
		}
	}

	return result, nil
}
