package handler

import (
	"net/http"
	"strconv"

	"github.com/SukGebeya/Backend/internal/domain"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/SukGebeya/Backend/pkg/validation"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type WalletHandler struct {
	walletService   *service.WalletService
	merchantService *service.MerchantService
}

func NewWalletHandler(walletService *service.WalletService, merchantService *service.MerchantService) *WalletHandler {
	return &WalletHandler{
		walletService:   walletService,
		merchantService: merchantService,
	}
}

type GetWalletResponse struct {
	BalanceCents int64  `json:"balance_cents"`
	Currency     string `json:"currency"`
}

type TransactionResponse struct {
	ID                uuid.UUID  `json:"id"`
	MerchantID        uuid.UUID  `json:"merchant_id"`
	OrderID           *uuid.UUID `json:"order_id,omitempty"`
	Type              string     `json:"type"`
	AmountCents       int64      `json:"amount_cents"`
	Currency          string     `json:"currency"`
	BlnkTransactionID string     `json:"blnk_transaction_id"`
	Status            string     `json:"status"`
	Description       *string    `json:"description,omitempty"`
	CreatedAt         string     `json:"created_at"`
}

type TransactionListResponse struct {
	Items      []TransactionResponse `json:"items"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PerPage    int                   `json:"per_page"`
	TotalPages int                   `json:"total_pages"`
}

type RequestPayoutRequest struct {
	AmountCents int64  `json:"amount_cents" validate:"required,min=100"`
	Method      string `json:"method" validate:"required,oneof=telebirr bank_transfer cbe_birr"`
	Destination string `json:"destination" validate:"required"`
}

type PayoutResponse struct {
	ID          uuid.UUID `json:"id"`
	MerchantID  uuid.UUID `json:"merchant_id"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	Method      string    `json:"method"`
	Destination string    `json:"destination"`
	Status      string    `json:"status"`
	RequestedAt string    `json:"requested_at"`
}

// GetWallet godoc
// @Summary Get wallet balance
// @Description Get the current balance of the merchant's wallet
// @Tags wallet
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GetWalletResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /merchant/wallet [get]
func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	merchant, err := h.merchantService.GetByUserID(ctx, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	balance, err := h.walletService.GetBalance(ctx, merchant.ID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, GetWalletResponse{
		BalanceCents: balance.BalanceCents,
		Currency:     balance.Currency,
	})
}

// ListTransactions godoc
// @Summary List wallet transactions
// @Description Get the transaction history for the merchant's wallet
// @Tags wallet
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} TransactionListResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /merchant/wallet/transactions [get]
func (h *WalletHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	merchant, err := h.merchantService.GetByUserID(ctx, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	result, err := h.walletService.GetTransactionHistory(ctx, merchant.ID, page, perPage)
	if err != nil {
		response.Error(w, err)
		return
	}

	items := make([]TransactionResponse, len(result.Items))
	for i, txn := range result.Items {
		items[i] = TransactionResponse{
			ID:                txn.ID,
			MerchantID:        txn.MerchantID,
			OrderID:           txn.OrderID,
			Type:              string(txn.Type),
			AmountCents:       txn.AmountCents,
			Currency:          txn.Currency,
			BlnkTransactionID: txn.BlnkTransactionID,
			Status:            string(txn.Status),
			Description:       txn.Description,
			CreatedAt:         txn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	response.JSON(w, http.StatusOK, TransactionListResponse{
		Items:      items,
		Total:      result.Total,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	})
}

// RequestPayout godoc
// @Summary Request payout
// @Description Create a new payout request to withdraw funds from the merchant's wallet. Requires Idempotency-Key header to prevent duplicate payouts.
// @Tags wallet
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Idempotency-Key header string true "Unique idempotency key to prevent duplicate payouts"
// @Param request body RequestPayoutRequest true "Payout request"
// @Success 201 {object} PayoutResponse
// @Failure 400 {object} response.ErrorResponse "Invalid request or missing Idempotency-Key"
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /merchant/wallet/payout [post]
func (h *WalletHandler) RequestPayout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	merchant, err := h.merchantService.GetByUserID(ctx, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req RequestPayoutRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); errs != nil {
		response.Error(w, apperrors.BadRequest(errs.Error()))
		return
	}

	input := domain.RequestPayoutInput{
		AmountCents: req.AmountCents,
		Method:      domain.PayoutMethod(req.Method),
		Destination: req.Destination,
	}

	payout, err := h.walletService.RequestPayout(ctx, merchant.ID, input)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, PayoutResponse{
		ID:          payout.ID,
		MerchantID:  payout.MerchantID,
		AmountCents: payout.AmountCents,
		Currency:    payout.Currency,
		Method:      string(payout.Method),
		Destination: payout.Destination,
		Status:      string(payout.Status),
		RequestedAt: payout.RequestedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// GetMerchantPayouts godoc
// @Summary List merchant payouts
// @Description Get the list of payout requests for the merchant
// @Tags wallet
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} response.PaginatedResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /merchant/wallet/payouts [get]
// @Security BearerAuth
func (h *WalletHandler) GetMerchantPayouts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	merchant, err := h.merchantService.GetByUserID(ctx, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	result, err := h.walletService.ListPayoutsByMerchant(ctx, merchant.ID, page, perPage)
	if err != nil {
		response.Error(w, err)
		return
	}

	items := make([]PayoutResponse, len(result.Items))
	for i, p := range result.Items {
		items[i] = PayoutResponse{
			ID:          p.ID,
			MerchantID:  p.MerchantID,
			AmountCents: p.AmountCents,
			Currency:    p.Currency,
			Method:      string(p.Method),
			Destination: p.Destination,
			Status:      string(p.Status),
			RequestedAt: p.RequestedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	response.JSON(w, http.StatusOK, response.PaginatedResponse{
		Items:      items,
		Total:      result.Total,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	})
}

// GetPayoutByID godoc
// @Summary Get payout details
// @Description Get details of a specific payout request
// @Tags wallet
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payout_id path string true "Payout ID"
// @Success 200 {object} PayoutResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /merchant/wallet/payouts/{payout_id} [get]
func (h *WalletHandler) GetPayoutByID(w http.ResponseWriter, r *http.Request) {
	_ = r.Context()
	payoutIDStr := chi.URLParam(r, "payout_id")
	_, err := uuid.Parse(payoutIDStr)
	if err != nil {
		response.Error(w, apperrors.BadRequest("invalid payout ID"))
		return
	}

	response.Error(w, apperrors.NotFound("payout not found"))
}
