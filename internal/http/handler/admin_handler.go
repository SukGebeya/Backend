package handler

import (
	"net/http"
	"strconv"

	"github.com/SukGebeya/Backend/internal/domain"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/SukGebeya/Backend/pkg/validation"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AdminHandler struct {
	walletService  *service.WalletService
	userService    *service.UserService
	merchantRepo   repository.MerchantRepository
	merchantSvc    *service.MerchantService
	orderService   *service.OrderService
	reconciliation *service.ReconciliationService
}

func NewAdminHandler(
	walletService *service.WalletService,
	userService *service.UserService,
	merchantRepo repository.MerchantRepository,
	merchantSvc *service.MerchantService,
	orderService *service.OrderService,
	reconciliation *service.ReconciliationService,
) *AdminHandler {
	return &AdminHandler{
		walletService:  walletService,
		userService:    userService,
		merchantRepo:   merchantRepo,
		merchantSvc:    merchantSvc,
		orderService:   orderService,
		reconciliation: reconciliation,
	}
}

type ResolveDisputeRequest struct {
	Outcome string `json:"outcome" validate:"required,oneof=release refund" example:"refund"`
	Restock bool   `json:"restock"`
}

// ResolveDispute godoc
// @Summary Resolve an order dispute (admin)
// @Description release pays the merchant as if delivery were confirmed; refund cancels the order and refunds the buyer through Chapa (set restock when the goods came back).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body ResolveDisputeRequest true "Resolution"
// @Success 200 {object} OrderResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Router /admin/orders/{id}/resolve [post]
func (h *AdminHandler) ResolveDispute(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	var req ResolveDisputeRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}
	order, err := h.orderService.ResolveDispute(r.Context(), orderID, service.DisputeOutcome(req.Outcome), req.Restock)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toOrderResponse(order))
}

// ProvisionWallet godoc
// @Summary Create a missing merchant wallet (admin)
// @Description Creates the merchant's Blnk wallet if it has none (seeded merchants, or onboarding while Blnk was down). Safe to repeat.
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "Merchant ID"
// @Success 200 {object} MerchantProfileResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /admin/merchants/{id}/wallet [post]
func (h *AdminHandler) ProvisionWallet(w http.ResponseWriter, r *http.Request) {
	merchantID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	merchant, err := h.merchantSvc.ProvisionWallet(r.Context(), merchantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toMerchantProfileResponse(merchant))
}

// Reconciliation godoc
// @Summary Reconcile ledger with database (admin)
// @Description Compares every merchant wallet and the escrow balance in Blnk with Postgres records. Also runs daily at 03:00 Addis Ababa time.
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} service.ReconciliationReport
// @Failure 500 {object} response.ErrorResponse
// @Router /admin/reconciliation [get]
func (h *AdminHandler) Reconciliation(w http.ResponseWriter, r *http.Request) {
	report, err := h.reconciliation.Run(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, report)
}

type RejectPayoutRequest struct {
	Reason string `json:"reason" validate:"max=500"`
}

type AdminPayoutResponse struct {
	ID                uuid.UUID `json:"id"`
	MerchantID        uuid.UUID `json:"merchant_id"`
	AmountCents       int64     `json:"amount_cents"`
	Currency          string    `json:"currency"`
	Method            string    `json:"method"`
	Destination       string    `json:"destination"`
	Status            string    `json:"status"`
	AccountName       *string   `json:"account_name,omitempty"`
	BankCode          *string   `json:"bank_code,omitempty"`
	BlnkTransactionID *string   `json:"blnk_transaction_id,omitempty"`
	TransferReference *string   `json:"transfer_reference,omitempty"`
	FailureReason     *string   `json:"failure_reason,omitempty"`
	RequestedAt       string    `json:"requested_at"`
	ProcessedAt       *string   `json:"processed_at,omitempty"`
}

type AdminPayoutListResponse struct {
	Items      []AdminPayoutResponse `json:"items"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PerPage    int                   `json:"per_page"`
	TotalPages int                   `json:"total_pages"`
}

func toAdminPayoutResponse(p *domain.Payout) AdminPayoutResponse {
	var processedAt *string
	if p.ProcessedAt != nil {
		formatted := p.ProcessedAt.Format("2006-01-02T15:04:05Z07:00")
		processedAt = &formatted
	}

	return AdminPayoutResponse{
		ID:                p.ID,
		MerchantID:        p.MerchantID,
		AmountCents:       p.AmountCents,
		Currency:          p.Currency,
		Method:            string(p.Method),
		Destination:       p.Destination,
		Status:            string(p.Status),
		AccountName:       p.AccountName,
		BankCode:          p.BankCode,
		BlnkTransactionID: p.BlnkTransactionID,
		TransferReference: p.TransferReference,
		FailureReason:     p.FailureReason,
		RequestedAt:       p.RequestedAt.Format("2006-01-02T15:04:05Z07:00"),
		ProcessedAt:       processedAt,
	}
}

// ListPayouts godoc
// @Summary List all payouts (admin)
// @Description Get a paginated list of all payout requests with optional status filter
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status (pending, processing, completed, failed)"
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} AdminPayoutListResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /admin/payouts [get]
func (h *AdminHandler) ListPayouts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	var statusFilter *domain.PayoutStatus
	if raw := r.URL.Query().Get("status"); raw != "" {
		st := domain.PayoutStatus(raw)
		statusFilter = &st
	}

	result, err := h.walletService.ListPayouts(ctx, statusFilter, page, perPage)
	if err != nil {
		response.Error(w, err)
		return
	}
	payouts, total := result.Items, result.Total
	if statusFilter == nil {
		total = int64(len(payouts))
	}

	items := make([]AdminPayoutResponse, len(payouts))
	for i, p := range payouts {
		items[i] = toAdminPayoutResponse(&p)
	}

	response.JSON(w, http.StatusOK, AdminPayoutListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: int((total + int64(perPage) - 1) / int64(perPage)),
	})
}

// ProcessPayout godoc
// @Summary Approve and process a payout (admin)
// @Description Approves a pending payout and starts the Chapa transfer asynchronously. The merchant's funds are already held, so approval cannot overdraw the wallet.
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Idempotency-Key header string false "Optional idempotency key to prevent duplicate processing"
// @Param id path string true "Payout ID"
// @Success 202 {object} AdminPayoutResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /admin/payouts/{id}/process [post]
func (h *AdminHandler) ProcessPayout(w http.ResponseWriter, r *http.Request) {
	payoutID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperrors.BadRequest("invalid payout ID"))
		return
	}

	payout, err := h.walletService.ApprovePayout(r.Context(), payoutID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, toAdminPayoutResponse(payout))
}

// RejectPayout godoc
// @Summary Reject a payout (admin)
// @Description Rejects a pending payout and releases the held funds back to the merchant's wallet
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Payout ID"
// @Param request body RejectPayoutRequest false "Rejection reason"
// @Success 200 {object} AdminPayoutResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /admin/payouts/{id}/reject [post]
func (h *AdminHandler) RejectPayout(w http.ResponseWriter, r *http.Request) {
	payoutID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperrors.BadRequest("invalid payout ID"))
		return
	}

	var req RejectPayoutRequest
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &req); err != nil {
			response.Error(w, err)
			return
		}
	}

	payout, err := h.walletService.RejectPayout(r.Context(), payoutID, req.Reason)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toAdminPayoutResponse(payout))
}

// ListUsers godoc
// @Summary List all users (admin)
// @Description Get a paginated list of all users
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} response.PaginatedResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /admin/users [get]
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	opts := parseListOptions(r)

	result, err := h.userService.List(ctx, opts)
	if err != nil {
		response.Error(w, err)
		return
	}

	users := make([]UserResponse, len(result.Items))
	for i, u := range result.Items {
		users[i] = *toUserResponse(&u)
	}

	response.JSON(w, http.StatusOK, response.PaginatedResponse{
		Items:      users,
		Total:      result.Total,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	})
}

// UpdateKYC godoc
// @Summary Update user KYC status (admin)
// @Description Update the KYC verification status of a user
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body UpdateKYCRequest true "KYC update request"
// @Success 200 {object} UserResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /admin/users/{id}/kyc [put]
func (h *AdminHandler) UpdateKYC(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	var req UpdateKYCRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	user, err := h.userService.UpdateKYCStatus(ctx, userID, domain.KYCStatus(req.KYCStatus))
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toUserResponse(user))
}

type UpdateKYCRequest struct {
	KYCStatus string `json:"kyc_status" validate:"required,oneof=pending submitted verified rejected"`
}

// ListMerchants godoc
// @Summary List all merchants (admin)
// @Description Get a paginated list of all merchant profiles
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} response.PaginatedResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /admin/merchants [get]
func (h *AdminHandler) ListMerchants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	opts := parseListOptions(r)

	result, err := h.merchantRepo.List(ctx, opts)
	if err != nil {
		response.Error(w, err)
		return
	}

	merchants := make([]MerchantResponse, len(result.Items))
	for i, m := range result.Items {
		merchants[i] = *toMerchantResponse(&m)
	}

	response.JSON(w, http.StatusOK, response.PaginatedResponse{
		Items:      merchants,
		Total:      result.Total,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	})
}

// VerifyMerchant godoc
// @Summary Verify a merchant (admin)
// @Description Set a merchant's verified status
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Merchant ID"
// @Param request body VerifyMerchantRequest true "Verification request"
// @Success 200 {object} MerchantResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /admin/merchants/{id}/verify [put]
func (h *AdminHandler) VerifyMerchant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	merchantID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	var req VerifyMerchantRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	merchant, err := h.merchantSvc.Verify(ctx, merchantID, req.IsVerified)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toMerchantResponse(merchant))
}

type VerifyMerchantRequest struct {
	IsVerified bool `json:"is_verified"`
}

type MerchantResponse struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	ShopName        string  `json:"shop_name"`
	ShopSlug        string  `json:"shop_slug"`
	ShopDescription *string `json:"shop_description,omitempty"`
	ShopLogoURL     *string `json:"shop_logo_url,omitempty"`
	ShopBannerURL   *string `json:"shop_banner_url,omitempty"`
	ContactPhone    string  `json:"contact_phone"`
	ContactEmail    *string `json:"contact_email,omitempty"`
	Address         *string `json:"address,omitempty"`
	City            string  `json:"city"`
	BlnkBalanceID   *string `json:"blnk_balance_id,omitempty"`
	IsVerified      bool    `json:"is_verified"`
	TotalSalesCents int64   `json:"total_sales_cents"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func toMerchantResponse(m *domain.MerchantProfile) *MerchantResponse {
	return &MerchantResponse{
		ID:              m.ID.String(),
		UserID:          m.UserID.String(),
		ShopName:        m.ShopName,
		ShopSlug:        m.ShopSlug,
		ShopDescription: m.ShopDescription,
		ShopLogoURL:     m.ShopLogoURL,
		ShopBannerURL:   m.ShopBannerURL,
		ContactPhone:    m.ContactPhone,
		ContactEmail:    m.ContactEmail,
		Address:         m.Address,
		City:            m.City,
		BlnkBalanceID:   m.BlnkBalanceID,
		IsVerified:      m.IsVerified,
		TotalSalesCents: m.TotalSalesCents,
		CreatedAt:       m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
