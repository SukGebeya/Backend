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
	productRepo   repository.ProductRepository
}

// WithCatalog lets cart views include product names and links.
func (s *CartService) WithCatalog(productRepo repository.ProductRepository) *CartService {
	s.productRepo = productRepo
	return s
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
		if live, err := s.checkExpiry(ctx, cart); err == nil {
			return live, nil
		}
	} else if !domainerrors.IsNotFound(err) {
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
		if live, err := s.checkExpiry(ctx, cart); err == nil {
			return live, nil
		}
	} else if !domainerrors.IsNotFound(err) {
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
		if err := s.expire(ctx, cart); err != nil {
			slog.Error("failed to expire cart", "cart_id", cart.ID, "error", err)
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

	existing, err := s.cartItemRepo.GetByCartAndVariant(ctx, cartID, variantID)
	if err != nil && !domainerrors.IsNotFound(err) {
		return nil, domainerrors.Internal("failed to check existing item", err)
	}

	if existing != nil {
		newQty := existing.Quantity + quantity
		if newQty > domain.MaxQuantityPerItem {
			return nil, domainerrors.BadRequest("maximum quantity per item is 100")
		}

		if err := s.reserve(ctx, variantID, quantity); err != nil {
			return nil, err
		}

		updated, err := s.cartItemRepo.UpdateQuantity(ctx, existing.ID, newQty)
		if err != nil {
			s.release(ctx, variantID, quantity)
			return nil, domainerrors.Internal("failed to update cart item", err)
		}
		return updated, nil
	}

	if quantity > domain.MaxQuantityPerItem {
		return nil, domainerrors.BadRequest("maximum quantity per item is 100")
	}

	count, err := s.cartItemRepo.CountDistinctVariants(ctx, cartID)
	if err != nil {
		return nil, domainerrors.Internal("failed to count cart items", err)
	}
	if count >= domain.MaxCartItems {
		return nil, domainerrors.BadRequest("cart cannot contain more than 50 different items")
	}

	if err := s.reserve(ctx, variantID, quantity); err != nil {
		return nil, err
	}

	created, err := s.cartItemRepo.Create(ctx, &domain.CartItem{
		ID:                 uuid.New(),
		CartID:             cartID,
		VariantID:          variantID,
		Quantity:           quantity,
		PriceSnapshotCents: priceCents,
	})
	if err != nil {
		s.release(ctx, variantID, quantity)
		return nil, domainerrors.Internal("failed to add item to cart", err)
	}

	return created, nil
}

// reserve holds stock for a cart line. A variant without an inventory record is
// treated as out of stock so nothing can be sold that was never stocked.
func (s *CartService) reserve(ctx context.Context, variantID uuid.UUID, qty int) error {
	if _, err := s.inventoryRepo.GetByVariantID(ctx, variantID); err != nil {
		if domainerrors.IsNotFound(err) {
			return domainerrors.InsufficientInventory("variant is out of stock")
		}
		return domainerrors.Internal("failed to check inventory", err)
	}
	return s.inventoryRepo.Reserve(ctx, variantID, qty)
}

func (s *CartService) release(ctx context.Context, variantID uuid.UUID, qty int) {
	if err := s.inventoryRepo.Release(ctx, variantID, qty); err != nil {
		slog.Error("failed to release inventory reservation", "variant_id", variantID, "quantity", qty, "error", err)
	}
}

func (s *CartService) UpdateItemQuantity(ctx context.Context, cartID, itemID uuid.UUID, quantity int) (*domain.CartItem, error) {
	item, err := s.cartItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}

	if item.CartID != cartID {
		return nil, domainerrors.Forbidden("item does not belong to this cart")
	}

	return s.UpdateItemQuantityByVariant(ctx, cartID, item.VariantID, quantity)
}

func (s *CartService) RemoveItem(ctx context.Context, cartID, itemID uuid.UUID) error {
	item, err := s.cartItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return err
	}

	if item.CartID != cartID {
		return domainerrors.Forbidden("item does not belong to this cart")
	}

	return s.RemoveItemByVariant(ctx, cartID, item.VariantID)
}

func (s *CartService) RemoveItemByVariant(ctx context.Context, cartID, variantID uuid.UUID) error {
	item, err := s.cartItemRepo.GetByCartAndVariant(ctx, cartID, variantID)
	if err != nil {
		return err
	}

	if err := s.cartItemRepo.Delete(ctx, item.ID); err != nil {
		return domainerrors.Internal("failed to remove item from cart", err)
	}

	s.release(ctx, variantID, item.Quantity)
	return nil
}

func (s *CartService) UpdateItemQuantityByVariant(ctx context.Context, cartID, variantID uuid.UUID, quantity int) (*domain.CartItem, error) {
	if quantity <= 0 {
		if err := s.RemoveItemByVariant(ctx, cartID, variantID); err != nil {
			return nil, err
		}
		return nil, nil
	}

	if quantity > domain.MaxQuantityPerItem {
		return nil, domainerrors.BadRequest("maximum quantity per item is 100")
	}

	item, err := s.cartItemRepo.GetByCartAndVariant(ctx, cartID, variantID)
	if err != nil {
		return nil, err
	}

	if quantity == item.Quantity {
		return item, nil
	}

	delta := quantity - item.Quantity
	if delta > 0 {
		if err := s.reserve(ctx, variantID, delta); err != nil {
			return nil, err
		}
	}

	updated, err := s.cartItemRepo.UpdateQuantity(ctx, item.ID, quantity)
	if err != nil {
		if delta > 0 {
			s.release(ctx, variantID, delta)
		}
		return nil, domainerrors.Internal("failed to update cart item quantity", err)
	}

	if delta < 0 {
		s.release(ctx, variantID, -delta)
	}

	return updated, nil
}

func (s *CartService) ClearCart(ctx context.Context, cartID uuid.UUID) error {
	items, err := s.cartItemRepo.ListByCart(ctx, cartID)
	if err != nil {
		return domainerrors.Internal("failed to list cart items", err)
	}

	if err := s.cartItemRepo.ClearCart(ctx, cartID); err != nil {
		return err
	}

	for _, item := range items {
		s.release(ctx, item.VariantID, item.Quantity)
	}
	return nil
}

// ExpireCart releases every reservation held by an active cart and marks it abandoned.
func (s *CartService) ExpireCart(ctx context.Context, cartID uuid.UUID) error {
	cart, err := s.cartRepo.GetByID(ctx, cartID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if cart.Status != domain.CartStatusActive {
		return nil
	}

	if time.Now().Before(cart.ExpiresAt) {
		return nil
	}

	return s.expire(ctx, cart)
}

func (s *CartService) expire(ctx context.Context, cart *domain.Cart) error {
	if err := s.ClearCart(ctx, cart.ID); err != nil {
		return err
	}
	if cart.SessionID != nil {
		if err := s.cartRepo.Delete(ctx, cart.ID); err != nil {
			return err
		}
	} else if _, err := s.cartRepo.UpdateStatus(ctx, cart.ID, domain.CartStatusAbandoned); err != nil {
		return err
	}
	slog.Info("cart expired", "cart_id", cart.ID)
	return nil
}

// MergeCart moves a guest cart into the user's cart. When both hold the same
// variant the larger quantity wins and the surplus reservation is released.
func (s *CartService) MergeCart(ctx context.Context, userID uuid.UUID, sessionID string) (*domain.Cart, error) {
	userCart, err := s.getOrCreateUserCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	sessionCart, err := s.cartRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return s.withItems(ctx, userCart)
		}
		return nil, domainerrors.Internal("failed to get session cart", err)
	}

	if sessionCart.ID == userCart.ID {
		return s.withItems(ctx, userCart)
	}

	sessionItems, err := s.cartItemRepo.ListByCart(ctx, sessionCart.ID)
	if err != nil {
		return nil, domainerrors.Internal("failed to list session cart items", err)
	}

	for _, item := range sessionItems {
		existing, err := s.cartItemRepo.GetByCartAndVariant(ctx, userCart.ID, item.VariantID)
		if err != nil && !domainerrors.IsNotFound(err) {
			return nil, domainerrors.Internal("failed to check user cart", err)
		}

		if existing == nil {
			if _, err := s.cartItemRepo.Create(ctx, &domain.CartItem{
				ID:                 uuid.New(),
				CartID:             userCart.ID,
				VariantID:          item.VariantID,
				Quantity:           item.Quantity,
				PriceSnapshotCents: item.PriceSnapshotCents,
			}); err != nil {
				return nil, domainerrors.Internal("failed to merge cart item", err)
			}
			continue
		}

		keep := max(existing.Quantity, item.Quantity)
		if keep != existing.Quantity {
			if _, err := s.cartItemRepo.UpdateQuantity(ctx, existing.ID, keep); err != nil {
				return nil, domainerrors.Internal("failed to merge cart item", err)
			}
		}
		s.release(ctx, item.VariantID, min(existing.Quantity, item.Quantity))
	}

	if err := s.cartItemRepo.ClearCart(ctx, sessionCart.ID); err != nil {
		return nil, err
	}
	if err := s.cartRepo.DeleteBySessionID(ctx, sessionID); err != nil {
		slog.Error("failed to delete session cart after merge", "session_id", sessionID, "error", err)
	}
	if err := s.sessionRepo.Delete(ctx, sessionID); err != nil {
		slog.Error("failed to delete session after merge", "session_id", sessionID, "error", err)
	}

	return s.withItems(ctx, userCart)
}

func (s *CartService) withItems(ctx context.Context, cart *domain.Cart) (*domain.Cart, error) {
	items, err := s.cartItemRepo.ListByCart(ctx, cart.ID)
	if err != nil {
		return nil, domainerrors.Internal("failed to get cart items", err)
	}
	cart.Items = items
	return cart, nil
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

		details := domain.CartItemWithDetails{
			CartItem:     item,
			VariantTitle: variant.Title,
			Currency:     variant.Currency,
		}
		if s.productRepo != nil {
			if product, err := s.productRepo.GetByID(ctx, variant.ProductID); err == nil {
				details.ProductName = product.Name
				details.ProductSlug = product.Slug
				details.MerchantID = product.MerchantID.String()
			}
		}
		result.Items[i] = details
	}

	return result, nil
}
