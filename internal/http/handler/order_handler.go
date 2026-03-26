package handler

import (
	"net/http"

	"github.com/SukGebeya/Backend/internal/domain"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/SukGebeya/Backend/pkg/validation"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

type OrderResponse struct {
	ID                string  `json:"id"`
	BuyerID           string  `json:"buyer_id"`
	MerchantID        string  `json:"merchant_id"`
	ShippingAddressID *string `json:"shipping_address_id,omitempty"`
	Status            string  `json:"status"`
	SubtotalCents     int64   `json:"subtotal_cents"`
	PlatformFeeCents  int64   `json:"platform_fee_cents"`
	TotalCents        int64   `json:"total_cents"`
	Currency          string  `json:"currency"`
	Notes             *string `json:"notes,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type OrderDetailResponse struct {
	Order OrderResponse       `json:"order"`
	Items []OrderItemResponse `json:"items"`
}

type OrderItemResponse struct {
	ID              string `json:"id"`
	VariantID       string `json:"variant_id"`
	ProductName     string `json:"product_name"`
	VariantTitle    string `json:"variant_title"`
	Quantity        int    `json:"quantity"`
	UnitPriceCents  int64  `json:"unit_price_cents"`
	TotalPriceCents int64  `json:"total_price_cents"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=pending_payment paid processing shipped delivered cancelled refunded" example:"processing"`
}

// ListMyOrders godoc
// @Summary List buyer's orders
// @Description Get a paginated list of orders for the authenticated buyer
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} response.PaginatedResponse "List of orders"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /orders [get]
func (h *OrderHandler) ListMyOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	opts := parseListOptions(r)

	result, err := h.orderService.ListByBuyerID(r.Context(), userID, opts.Page, opts.PerPage)
	if err != nil {
		response.Error(w, err)
		return
	}

	orders := make([]OrderResponse, len(result.Items))
	for i, order := range result.Items {
		orders[i] = toOrderResponse(&order)
	}

	response.JSON(w, http.StatusOK, response.PaginatedResponse{
		Items:      orders,
		Total:      result.Total,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	})
}

// GetOrder godoc
// @Summary Get order by ID
// @Description Get a specific order with its items
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Success 200 {object} OrderDetailResponse "Order details"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Order not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /orders/{id} [get]
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	orderWithItems, err := h.orderService.GetOrderWithItems(r.Context(), orderID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if orderWithItems.Order.BuyerID != userID {
		response.Error(w, forbidden("not authorized to view this order"))
		return
	}

	response.JSON(w, http.StatusOK, toOrderDetailResponse(orderWithItems))
}

// ConfirmDelivery godoc
// @Summary Confirm order delivery
// @Description Buyer confirms delivery of an order (triggers escrow release)
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Success 200 {object} OrderResponse "Updated order"
// @Failure 400 {object} response.ErrorResponse "Order cannot be confirmed"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Order not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /orders/{id}/confirm-delivery [post]
func (h *OrderHandler) ConfirmDelivery(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	order, err := h.orderService.GetByID(r.Context(), orderID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if order.BuyerID != userID {
		response.Error(w, forbidden("not authorized to confirm this order"))
		return
	}

	updated, err := h.orderService.ConfirmDelivery(r.Context(), orderID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toOrderResponse(updated))
}

// ListMerchantOrders godoc
// @Summary List merchant's orders
// @Description Get a paginated list of orders for the authenticated merchant
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param status query string false "Filter by status"
// @Success 200 {object} response.PaginatedResponse "List of orders"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /merchant/orders [get]
func (h *OrderHandler) ListMerchantOrders(w http.ResponseWriter, r *http.Request) {
	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
		return
	}

	opts := parseListOptions(r)

	result, err := h.orderService.ListByMerchantID(r.Context(), merchant.ID, opts.Page, opts.PerPage)
	if err != nil {
		response.Error(w, err)
		return
	}

	orders := make([]OrderResponse, len(result.Items))
	for i, order := range result.Items {
		orders[i] = toOrderResponse(&order)
	}

	response.JSON(w, http.StatusOK, response.PaginatedResponse{
		Items:      orders,
		Total:      result.Total,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	})
}

// UpdateOrderStatus godoc
// @Summary Update order status (merchant)
// @Description Merchant updates order status (e.g., processing, shipped)
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body UpdateOrderStatusRequest true "Status update"
// @Success 200 {object} OrderResponse "Updated order"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Order not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /merchant/orders/{id}/status [put]
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
		return
	}

	var req UpdateOrderStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	order, err := h.orderService.GetByID(r.Context(), orderID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if order.MerchantID != merchant.ID {
		response.Error(w, forbidden("not authorized to update this order"))
		return
	}

	status := domain.OrderStatus(req.Status)
	if !status.IsValid() {
		response.Error(w, apperrors.BadRequest("invalid order status"))
		return
	}

	updated, err := h.orderService.UpdateStatus(r.Context(), orderID, status)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toOrderResponse(updated))
}

func toOrderResponse(o *domain.Order) OrderResponse {
	var shippingAddressID *string
	if o.ShippingAddressID != nil {
		id := o.ShippingAddressID.String()
		shippingAddressID = &id
	}

	return OrderResponse{
		ID:                o.ID.String(),
		BuyerID:           o.BuyerID.String(),
		MerchantID:        o.MerchantID.String(),
		ShippingAddressID: shippingAddressID,
		Status:            string(o.Status),
		SubtotalCents:     o.SubtotalCents,
		PlatformFeeCents:  o.PlatformFeeCents,
		TotalCents:        o.TotalCents,
		Currency:          o.Currency,
		Notes:             o.Notes,
		CreatedAt:         o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:         o.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toOrderDetailResponse(o *domain.OrderWithItems) OrderDetailResponse {
	items := make([]OrderItemResponse, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItemResponse{
			ID:              item.ID.String(),
			VariantID:       item.VariantID.String(),
			ProductName:     item.ProductName,
			VariantTitle:    item.VariantTitle,
			Quantity:        item.Quantity,
			UnitPriceCents:  item.UnitPriceCents,
			TotalPriceCents: item.TotalPriceCents,
		}
	}

	return OrderDetailResponse{
		Order: toOrderResponse(&o.Order),
		Items: items,
	}
}
