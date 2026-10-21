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
	BalanceCents       int64  `json:"balance_cents"`
	PendingPayoutCents int64  `json:"pending_payout_cents"`
	AvailableCents     int64  `json:"available_cents"`
	Currency           string `json:"currency"`
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
	AmountCents int64  `json:"amount_cents" validate:"required,min=100" example:"50000"`
	Method      string `json:"method" validate:"required,oneof=telebirr bank_transfer cbe_birr" example:"telebirr"`
	Destination string `json:"destination" validate:"required,min=5,max=50" example:"0911234567"`
	AccountName string `json:"account_name" validate:"required,min=2,max=100" example:"Tigist Bekele"`
	BankCode    string `json:"bank_code" validate:"required,max=20" example:"855"`
}

type PayoutResponse struct {
	ID            uuid.UUID `json:"id"`
	MerchantID    uuid.UUID `json:"merchant_id"`
	AmountCents   int64     `json:"amount_cents"`
	Currency      string    `json:"currency"`
	Method        string    `json:"method"`
	Destination   string    `json:"destination"`
	AccountName   *string   `json:"account_name,omitempty"`
	BankCode      *string   `json:"bank_code,omitempty"`
	Status        string    `json:"status"`
	FailureReason *string   `json:"failure_reason,omitempty"`
	RequestedAt   string    `json:"requested_at"`
	ProcessedAt   *string   `json:"processed_at,omitempty"`
}

type BankResponse struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	IsMobile bool   `json:"is_mobile_money"`
}

func toPayoutResponse(p *domain.Payout) PayoutResponse {
	resp := PayoutResponse{
		ID:            p.ID,
		MerchantID:    p.MerchantID,
		AmountCents:   p.AmountCents,
		Currency:      p.Currency,
		Method:        string(p.Method),
		Destination:   p.Destination,
		AccountName:   p.AccountName,
		BankCode:      p.BankCode,
		Status:        string(p.Status),
		FailureReason: p.FailureReason,
		RequestedAt:   p.RequestedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if p.ProcessedAt != nil {
		processed := p.ProcessedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.ProcessedAt = &processed
	}
	return resp
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
		BalanceCents:       balance.BalanceCents,
		PendingPayoutCents: balance.PendingPayoutCents,
		AvailableCents:     balance.AvailableCents,
		Currency:           balance.Currency,
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
		AccountName: req.AccountName,
		BankCode:    req.BankCode,
	}

	payout, err := h.walletService.RequestPayout(ctx, merchant.ID, input)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, toPayoutResponse(payout))
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
	for i := range result.Items {
		items[i] = toPayoutResponse(&result.Items[i])
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
	payoutID, err := uuid.Parse(chi.URLParam(r, "payout_id"))
	if err != nil {
		response.Error(w, apperrors.BadRequest("invalid payout ID"))
		return
	}

	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, apperrors.Forbidden("merchant profile required"))
		return
	}

	payout, err := h.walletService.GetPayout(r.Context(), merchant.ID, payoutID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toPayoutResponse(payout))
}

// ListBanks godoc
// @Summary List payout banks
// @Description Banks and mobile-money providers Chapa can transfer to. Use the id as bank_code when requesting a payout.
// @Tags wallet
// @Produce json
// @Security BearerAuth
// @Success 200 {array} BankResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /merchant/wallet/banks [get]
func (h *WalletHandler) ListBanks(w http.ResponseWriter, r *http.Request) {
	banks, err := h.walletService.ListBanks(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}

	items := make([]BankResponse, len(banks))
	for i, b := range banks {
		items[i] = BankResponse{
			ID:       b.ID.String(),
			Slug:     b.Slug,
			Name:     b.Name,
			Currency: b.Currency,
			IsMobile: b.IsMobile == 1,
		}
	}

	response.JSON(w, http.StatusOK, items)
}
