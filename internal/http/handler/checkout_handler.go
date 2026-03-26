package handler

import (
	"net/http"

	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/SukGebeya/Backend/pkg/validation"
	"github.com/google/uuid"
)

type CheckoutHandler struct {
	checkoutService *service.CheckoutService
}

func NewCheckoutHandler(checkoutService *service.CheckoutService) *CheckoutHandler {
	return &CheckoutHandler{
		checkoutService: checkoutService,
	}
}

type CheckoutRequest struct {
	ShippingAddressID *string `json:"shipping_address_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Notes             *string `json:"notes" validate:"omitempty,max=500" example:"Please leave at the door"`
}

type CheckoutResponse struct {
	OrderID    string `json:"order_id"`
	TotalCents int64  `json:"total_cents"`
	Currency   string `json:"currency"`
	PaymentURL string `json:"payment_url"`
	ChapaTxRef string `json:"chapa_tx_ref"`
}

// Checkout godoc
// @Summary Process checkout
// @Description Create an order from cart and initiate payment
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CheckoutRequest true "Checkout details"
// @Success 201 {object} CheckoutResponse "Checkout result with payment URL"
// @Failure 400 {object} response.ErrorResponse "Invalid request or empty cart"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /orders/checkout [post]
func (h *CheckoutHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	var req CheckoutRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	var shippingAddressID *uuid.UUID
	if req.ShippingAddressID != nil {
		id, err := uuid.Parse(*req.ShippingAddressID)
		if err != nil {
			response.Error(w, apperrors.BadRequest("invalid shipping_address_id"))
			return
		}
		shippingAddressID = &id
	}

	result, err := h.checkoutService.Checkout(r.Context(), service.CheckoutInput{
		UserID:            userID,
		ShippingAddressID: shippingAddressID,
		Notes:             req.Notes,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, CheckoutResponse{
		OrderID:    result.OrderID.String(),
		TotalCents: result.TotalCents,
		Currency:   result.Currency,
		PaymentURL: result.PaymentURL,
		ChapaTxRef: result.ChapaTxRef,
	})
}
