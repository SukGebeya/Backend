package handler

import (
	"net/http"

	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
)

type DashboardHandler struct {
	orderService  *service.OrderService
	walletService *service.WalletService
}

func NewDashboardHandler(orderService *service.OrderService, walletService *service.WalletService) *DashboardHandler {
	return &DashboardHandler{orderService: orderService, walletService: walletService}
}

type MerchantDashboardResponse struct {
	ShopName           string           `json:"shop_name"`
	IsVerified         bool             `json:"is_verified"`
	TotalSalesCents    int64            `json:"total_sales_cents"`
	OrdersByStatus     map[string]int64 `json:"orders_by_status"`
	ActionRequired     int64            `json:"orders_action_required"`
	BalanceCents       int64            `json:"balance_cents"`
	AvailableCents     int64            `json:"available_cents"`
	PendingPayoutCents int64            `json:"pending_payout_cents"`
	Currency           string           `json:"currency"`
}

// GetDashboard godoc
// @Summary Merchant dashboard
// @Description Shop summary: total sales, orders by status (paid and processing need action), wallet balance and money held for pending payouts.
// @Tags merchant
// @Produce json
// @Security BearerAuth
// @Success 200 {object} MerchantDashboardResponse
// @Failure 403 {object} response.ErrorResponse
// @Router /merchant/dashboard [get]
func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
		return
	}

	counts, err := h.orderService.MerchantOrderCounts(r.Context(), merchant.ID)
	if err != nil {
		response.Error(w, err)
		return
	}
	balance, err := h.walletService.GetBalance(r.Context(), merchant.ID)
	if err != nil {
		response.Error(w, err)
		return
	}

	byStatus := make(map[string]int64, len(counts))
	for status, n := range counts {
		byStatus[string(status)] = n
	}

	response.JSON(w, http.StatusOK, MerchantDashboardResponse{
		ShopName:           merchant.ShopName,
		IsVerified:         merchant.IsVerified,
		TotalSalesCents:    merchant.TotalSalesCents,
		OrdersByStatus:     byStatus,
		ActionRequired:     counts[domain.OrderStatusPaid] + counts[domain.OrderStatusProcessing],
		BalanceCents:       balance.BalanceCents,
		AvailableCents:     balance.AvailableCents,
		PendingPayoutCents: balance.PendingPayoutCents,
		Currency:           balance.Currency,
	})
}
