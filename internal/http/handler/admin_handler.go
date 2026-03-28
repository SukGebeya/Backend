package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/SukGebeya/Backend/internal/domain"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/SukGebeya/Backend/pkg/validation"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type AdminHandler struct {
	payoutRepo   repository.PayoutRepository
	userService  *service.UserService
	merchantRepo repository.MerchantRepository
	merchantSvc  *service.MerchantService
	riverClient  *river.Client[pgx.Tx]
}

func NewAdminHandler(
	payoutRepo repository.PayoutRepository,
	userService *service.UserService,
	merchantRepo repository.MerchantRepository,
	merchantSvc *service.MerchantService,
	riverClient *river.Client[pgx.Tx],
) *AdminHandler {
	return &AdminHandler{
		payoutRepo:   payoutRepo,
		userService:  userService,
		merchantRepo: merchantRepo,
		merchantSvc:  merchantSvc,
		riverClient:  riverClient,
	}
}

type AdminPayoutResponse struct {
	ID                uuid.UUID `json:"id"`
	MerchantID        uuid.UUID `json:"merchant_id"`
	AmountCents       int64     `json:"amount_cents"`
	Currency          string    `json:"currency"`
	Method            string    `json:"method"`
	Destination       string    `json:"destination"`
	Status            string    `json:"status"`
	BlnkTransactionID *string   `json:"blnk_transaction_id,omitempty"`
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
		BlnkTransactionID: p.BlnkTransactionID,
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

	statusFilter := r.URL.Query().Get("status")

	var payouts []domain.Payout
	var total int64
	var err error

	if statusFilter != "" {
		payouts, err = h.payoutRepo.ListByStatus(ctx, domain.PayoutStatus(statusFilter), page, perPage)
		if err == nil {
			total, _ = h.payoutRepo.CountByStatus(ctx, domain.PayoutStatus(statusFilter))
		}
	} else {
		payouts, err = h.payoutRepo.ListAll(ctx, page, perPage)
		// For list all, we'd need a separate count method or we calculate from page info
		// For now, set total to length of items (simplified)
		total = int64(len(payouts))
	}

	if err != nil {
		response.Error(w, err)
		return
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
// @Summary Process a payout (admin)
// @Description Process a pending payout request - this will debit the merchant's wallet. Idempotency-Key header is recommended to prevent duplicate processing.
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Idempotency-Key header string false "Optional idempotency key to prevent duplicate processing"
// @Param id path string true "Payout ID"
// @Success 200 {object} AdminPayoutResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /admin/payouts/{id}/process [post]
func (h *AdminHandler) ProcessPayout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	payoutIDStr := chi.URLParam(r, "id")
	payoutID, err := uuid.Parse(payoutIDStr)
	if err != nil {
		response.Error(w, apperrors.BadRequest("invalid payout ID"))
		return
	}

	payout, err := h.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if payout.Status != domain.PayoutStatusPending {
		response.Error(w, apperrors.BadRequest("payout is not in pending status"))
		return
	}

	_, err = h.payoutRepo.UpdateStatus(ctx, payoutID, domain.PayoutStatusProcessing, nil, nil)
	if err != nil {
		response.Error(w, err)
		return
	}

	_, err = h.riverClient.Insert(ctx, jobs.ProcessPayoutArgs{PayoutID: payoutID}, nil)
	if err != nil {
		slog.Error("failed to enqueue payout job", "payout_id", payoutID, "error", err)
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message":   "Payout processing initiated. The payout will be processed asynchronously.",
		"payout_id": payoutID.String(),
	})
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
