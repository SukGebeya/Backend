package handler

import (
	"net/http"

	"github.com/SukGebeya/Backend/internal/domain"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/SukGebeya/Backend/pkg/validation"
	"github.com/google/uuid"
)

type CartHandler struct {
	cartService *service.CartService
	variantRepo repository.ProductVariantRepository
}

func NewCartHandler(cartService *service.CartService, variantRepo repository.ProductVariantRepository) *CartHandler {
	return &CartHandler{
		cartService: cartService,
		variantRepo: variantRepo,
	}
}

type AddToCartRequest struct {
	VariantID string `json:"variant_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Quantity  int    `json:"quantity" validate:"required,min=1,max=100" example:"2"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" validate:"required,min=0,max=100" example:"3"`
}

type CartItemResponse struct {
	ID                 string `json:"id"`
	CartID             string `json:"cart_id"`
	VariantID          string `json:"variant_id"`
	ProductName        string `json:"product_name"`
	VariantTitle       string `json:"variant_title"`
	Quantity           int    `json:"quantity"`
	PriceSnapshotCents int64  `json:"price_snapshot_cents"`
	Currency           string `json:"currency"`
	SubtotalCents      int64  `json:"subtotal_cents"`
}

type CartResponse struct {
	ID         string             `json:"id"`
	Status     string             `json:"status"`
	ExpiresAt  string             `json:"expires_at"`
	Items      []CartItemResponse `json:"items"`
	ItemCount  int                `json:"item_count"`
	TotalCents int64              `json:"total_cents"`
	Currency   string             `json:"currency"`
}

// GetCart godoc
// @Summary Get current cart
// @Description Get the current user's or session's cart with items
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} CartResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /cart [get]
func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	userID, hasUserID := middleware.GetUserID(r.Context())
	var sessionID *string
	if !hasUserID {
		sid, hasSessionID := middleware.GetSessionID(r.Context())
		if !hasSessionID {
			response.Error(w, apperrors.Unauthorized("authentication or session required"))
			return
		}
		sessionID = &sid
	}

	var uid *uuid.UUID
	if hasUserID {
		uid = &userID
	}

	cart, err := h.cartService.GetOrCreateCart(r.Context(), uid, sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}

	cartWithItems, err := h.cartService.GetCartWithItems(r.Context(), cart.ID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toCartResponse(cartWithItems))
}

// AddItem godoc
// @Summary Add item to cart
// @Description Add an item to the cart
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AddToCartRequest true "Add to cart request"
// @Success 200 {object} CartItemResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /cart/items [post]
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	userID, hasUserID := middleware.GetUserID(r.Context())
	var sessionID *string
	if !hasUserID {
		sid, hasSessionID := middleware.GetSessionID(r.Context())
		if !hasSessionID {
			response.Error(w, apperrors.Unauthorized("authentication or session required"))
			return
		}
		sessionID = &sid
	}

	var uid *uuid.UUID
	if hasUserID {
		uid = &userID
	}

	cart, err := h.cartService.GetOrCreateCart(r.Context(), uid, sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req AddToCartRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	variantID, err := uuid.Parse(req.VariantID)
	if err != nil {
		response.Error(w, apperrors.BadRequest("invalid variant_id"))
		return
	}

	variant, err := h.variantRepo.GetByID(r.Context(), variantID)
	if err != nil {
		response.Error(w, apperrors.NotFound("variant not found"))
		return
	}

	if !variant.IsActive {
		response.Error(w, apperrors.BadRequest("variant is not available"))
		return
	}

	item, err := h.cartService.AddItem(r.Context(), cart.ID, variantID, req.Quantity, variant.PriceCents)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toCartItemResponse(item, variant.Title, variant.Currency))
}

// UpdateItem godoc
// @Summary Update cart item quantity
// @Description Update the quantity of an item in the cart
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param variantId path string true "Variant ID"
// @Param request body UpdateCartItemRequest true "Update quantity request"
// @Success 200 {object} CartItemResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /cart/items/{variantId} [put]
func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	variantID, err := parseUUIDParam(r, "variantId")
	if err != nil {
		response.Error(w, err)
		return
	}

	userID, hasUserID := middleware.GetUserID(r.Context())
	var sessionID *string
	if !hasUserID {
		sid, hasSessionID := middleware.GetSessionID(r.Context())
		if !hasSessionID {
			response.Error(w, apperrors.Unauthorized("authentication or session required"))
			return
		}
		sessionID = &sid
	}

	var uid *uuid.UUID
	if hasUserID {
		uid = &userID
	}

	cart, err := h.cartService.GetOrCreateCart(r.Context(), uid, sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req UpdateCartItemRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	item, err := h.cartService.UpdateItemQuantityByVariant(r.Context(), cart.ID, variantID, req.Quantity)
	if err != nil {
		response.Error(w, err)
		return
	}

	if item == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	variant, err := h.variantRepo.GetByID(r.Context(), item.VariantID)
	if err != nil {
		response.Error(w, apperrors.Internal("failed to get variant", err))
		return
	}

	response.JSON(w, http.StatusOK, toCartItemResponse(item, variant.Title, variant.Currency))
}

// RemoveItem godoc
// @Summary Remove item from cart
// @Description Remove an item from the cart
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param variantId path string true "Variant ID"
// @Success 204 "No content"
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /cart/items/{variantId} [delete]
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	variantID, err := parseUUIDParam(r, "variantId")
	if err != nil {
		response.Error(w, err)
		return
	}

	userID, hasUserID := middleware.GetUserID(r.Context())
	var sessionID *string
	if !hasUserID {
		sid, hasSessionID := middleware.GetSessionID(r.Context())
		if !hasSessionID {
			response.Error(w, apperrors.Unauthorized("authentication or session required"))
			return
		}
		sessionID = &sid
	}

	var uid *uuid.UUID
	if hasUserID {
		uid = &userID
	}

	cart, err := h.cartService.GetOrCreateCart(r.Context(), uid, sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if err := h.cartService.RemoveItemByVariant(r.Context(), cart.ID, variantID); err != nil {
		response.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ClearCart godoc
// @Summary Clear cart
// @Description Remove all items from the cart
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 204 "No content"
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /cart [delete]
func (h *CartHandler) ClearCart(w http.ResponseWriter, r *http.Request) {
	userID, hasUserID := middleware.GetUserID(r.Context())
	var sessionID *string
	if !hasUserID {
		sid, hasSessionID := middleware.GetSessionID(r.Context())
		if !hasSessionID {
			response.Error(w, apperrors.Unauthorized("authentication or session required"))
			return
		}
		sessionID = &sid
	}

	var uid *uuid.UUID
	if hasUserID {
		uid = &userID
	}

	cart, err := h.cartService.GetOrCreateCart(r.Context(), uid, sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if err := h.cartService.ClearCart(r.Context(), cart.ID); err != nil {
		response.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MergeCart godoc
// @Summary Merge guest cart into user cart
// @Description Merge the guest cart associated with the session into the authenticated user's cart
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-Session-ID header string true "Session ID"
// @Success 200 {object} CartResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /cart/merge [post]
func (h *CartHandler) MergeCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	sessionID, ok := middleware.GetSessionID(r.Context())
	if !ok {
		response.Error(w, apperrors.BadRequest("session_id required"))
		return
	}

	cart, err := h.cartService.MergeCart(r.Context(), userID, sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}

	cartWithItems, err := h.cartService.GetCartWithItems(r.Context(), cart.ID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toCartResponse(cartWithItems))
}

func toCartResponse(cart *domain.CartWithDetails) *CartResponse {
	var totalCents int64
	items := make([]CartItemResponse, len(cart.Items))

	for i, item := range cart.Items {
		subtotal := item.PriceSnapshotCents * int64(item.Quantity)
		totalCents += subtotal

		items[i] = CartItemResponse{
			ID:                 item.ID.String(),
			CartID:             item.CartID.String(),
			VariantID:          item.VariantID.String(),
			ProductName:        item.ProductName,
			VariantTitle:       item.VariantTitle,
			Quantity:           item.Quantity,
			PriceSnapshotCents: item.PriceSnapshotCents,
			Currency:           item.Currency,
			SubtotalCents:      subtotal,
		}
	}

	currency := "ETB"
	if len(items) > 0 {
		currency = items[0].Currency
	}

	return &CartResponse{
		ID:         cart.ID.String(),
		Status:     string(cart.Status),
		ExpiresAt:  cart.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		Items:      items,
		ItemCount:  len(items),
		TotalCents: totalCents,
		Currency:   currency,
	}
}

func toCartItemResponse(item *domain.CartItem, variantTitle, currency string) *CartItemResponse {
	return &CartItemResponse{
		ID:                 item.ID.String(),
		CartID:             item.CartID.String(),
		VariantID:          item.VariantID.String(),
		VariantTitle:       variantTitle,
		Quantity:           item.Quantity,
		PriceSnapshotCents: item.PriceSnapshotCents,
		Currency:           currency,
		SubtotalCents:      item.PriceSnapshotCents * int64(item.Quantity),
	}
}
