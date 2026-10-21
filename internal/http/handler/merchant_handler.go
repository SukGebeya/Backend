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

type MerchantHandler struct {
	merchantService *service.MerchantService
}

func NewMerchantHandler(merchantService *service.MerchantService) *MerchantHandler {
	return &MerchantHandler{merchantService: merchantService}
}

type OnboardRequest struct {
	ShopName        string `json:"shop_name" validate:"required,min=2,max=100" example:"Tigist Habesha Designs"`
	ShopSlug        string `json:"shop_slug" validate:"omitempty,min=2,max=100,slug" example:"tigist-habesha"`
	ShopDescription string `json:"shop_description" validate:"omitempty,max=500" example:"Traditional Ethiopian clothing handmade in Addis"`
	ContactPhone    string `json:"contact_phone" validate:"required,etphone" example:"0911234567"`
	ContactEmail    string `json:"contact_email" validate:"omitempty,email" example:"tigist@example.com"`
	Address         string `json:"address" validate:"omitempty,max=200" example:"Bole, Addis Ababa"`
	City            string `json:"city" validate:"required,max=100" example:"Addis Ababa"`

	DeliveryFeeCents           int64   `json:"delivery_fee_cents" validate:"min=0,max=10000000" example:"15000"`
	FreeDeliveryThresholdCents *int64  `json:"free_delivery_threshold_cents" validate:"omitempty,min=0" example:"500000"`
	TIN                        *string `json:"tin" validate:"omitempty,numeric,len=10" example:"0012345678"`
	TradeLicenseNumber         *string `json:"trade_license_number" validate:"omitempty,max=50" example:"AA/12/3456/14"`
}

type OnboardResponse struct {
	MerchantID    string  `json:"merchant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ShopSlug      string  `json:"shop_slug" example:"tigist-habesha"`
	ShopName      string  `json:"shop_name" example:"Tigist Habesha Designs"`
	BlnkBalanceID *string `json:"blnk_balance_id,omitempty" example:"bln_0b7a3c1e-0000-0000-0000-000000000000"`
	Message       string  `json:"message" example:"Merchant profile created. KYC verification pending."`
}

type MerchantProfileResponse struct {
	ID              string `json:"id"`
	UserID          string `json:"user_id"`
	ShopName        string `json:"shop_name"`
	ShopSlug        string `json:"shop_slug"`
	ShopDescription string `json:"shop_description,omitempty"`
	ShopLogoURL     string `json:"shop_logo_url,omitempty"`
	ShopBannerURL   string `json:"shop_banner_url,omitempty"`
	ContactPhone    string `json:"contact_phone"`
	ContactEmail    string `json:"contact_email,omitempty"`
	Address         string `json:"address,omitempty"`
	City            string `json:"city"`
	IsVerified      bool   `json:"is_verified"`
	TotalSalesCents int64  `json:"total_sales_cents"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`

	DeliveryFeeCents           int64   `json:"delivery_fee_cents"`
	FreeDeliveryThresholdCents *int64  `json:"free_delivery_threshold_cents,omitempty"`
	TIN                        *string `json:"tin,omitempty"`
	TradeLicenseNumber         *string `json:"trade_license_number,omitempty"`
	HasWallet                  bool    `json:"has_wallet"`
}

type UpdateMerchantRequest struct {
	ShopName        string `json:"shop_name" validate:"omitempty,min=2,max=100"`
	ShopDescription string `json:"shop_description" validate:"omitempty,max=500"`
	ShopLogoURL     string `json:"shop_logo_url" validate:"omitempty,url"`
	ShopBannerURL   string `json:"shop_banner_url" validate:"omitempty,url"`
	ContactPhone    string `json:"contact_phone" validate:"omitempty,etphone"`
	ContactEmail    string `json:"contact_email" validate:"omitempty,email"`
	Address         string `json:"address" validate:"omitempty,max=200"`
	City            string `json:"city" validate:"omitempty,max=100"`

	DeliveryFeeCents           *int64  `json:"delivery_fee_cents" validate:"omitempty,min=0,max=10000000"`
	FreeDeliveryThresholdCents *int64  `json:"free_delivery_threshold_cents" validate:"omitempty,min=0"`
	ClearFreeDeliveryThreshold bool    `json:"clear_free_delivery_threshold"`
	TIN                        *string `json:"tin" validate:"omitempty,numeric,len=10"`
	TradeLicenseNumber         *string `json:"trade_license_number" validate:"omitempty,max=50"`
}

type ShopResponse struct {
	ShopSlug     string `json:"shop_slug"`
	ShopName     string `json:"shop_name"`
	Description  string `json:"description,omitempty"`
	LogoURL      string `json:"logo_url,omitempty"`
	BannerURL    string `json:"banner_url,omitempty"`
	ContactPhone string `json:"contact_phone"`
	ContactEmail string `json:"contact_email,omitempty"`
	City         string `json:"city"`
	IsVerified   bool   `json:"is_verified"`

	DeliveryFeeCents           int64  `json:"delivery_fee_cents"`
	FreeDeliveryThresholdCents *int64 `json:"free_delivery_threshold_cents,omitempty"`
}

// Onboard godoc
// @Summary Onboard a new merchant
// @Description Create a merchant profile for the authenticated user. Any authenticated user can onboard to become a merchant.
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body OnboardRequest true "Merchant onboarding details"
// @Success 201 {object} OnboardResponse "Merchant profile created"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or validation error"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 409 {object} response.ErrorResponse "User already has a merchant profile"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /merchant/onboard [post]
func (h *MerchantHandler) Onboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	var req OnboardRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	var shopDesc *string
	if req.ShopDescription != "" {
		shopDesc = &req.ShopDescription
	}
	var contactEmail *string
	if req.ContactEmail != "" {
		contactEmail = &req.ContactEmail
	}
	var address *string
	if req.Address != "" {
		address = &req.Address
	}

	merchant, err := h.merchantService.Onboard(r.Context(), service.OnboardInput{
		UserID:          userID,
		ShopName:        req.ShopName,
		ShopSlug:        req.ShopSlug,
		ShopDescription: shopDesc,
		ContactPhone:    req.ContactPhone,
		ContactEmail:    contactEmail,
		Address:         address,
		City:            req.City,

		DeliveryFeeCents:           req.DeliveryFeeCents,
		FreeDeliveryThresholdCents: req.FreeDeliveryThresholdCents,
		TIN:                        req.TIN,
		TradeLicenseNumber:         req.TradeLicenseNumber,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, OnboardResponse{
		MerchantID:    merchant.ID.String(),
		ShopSlug:      merchant.ShopSlug,
		ShopName:      merchant.ShopName,
		BlnkBalanceID: merchant.BlnkBalanceID,
		Message:       "Merchant profile created. KYC verification pending.",
	})
}

// GetProfile godoc
// @Summary Get merchant profile
// @Description Get the merchant profile for the authenticated user.
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} MerchantProfileResponse "Merchant profile"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 403 {object} response.ErrorResponse "Merchant profile required"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /merchant/profile [get]
func (h *MerchantHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, apperrors.Forbidden("merchant profile required"))
		return
	}

	response.JSON(w, http.StatusOK, toMerchantProfileResponse(merchant))
}

// Update Profile godoc
// @Summary Update merchant profile
// @Description Update the merchant profile for the authenticated user.
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateMerchantRequest true "Merchant profile update"
// @Success 200 {object} MerchantProfileResponse "Updated merchant profile"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or validation error"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 403 {object} response.ErrorResponse "Merchant profile required"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /merchant/profile [put]
func (h *MerchantHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, apperrors.Forbidden("merchant profile required"))
		return
	}

	var req UpdateMerchantRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	var shopName, contactPhone, city *string
	var shopDesc, shopLogoURL, shopBannerURL, contactEmail, address *string

	if req.ShopName != "" {
		shopName = &req.ShopName
	}
	if req.ShopDescription != "" {
		shopDesc = &req.ShopDescription
	}
	if req.ShopLogoURL != "" {
		shopLogoURL = &req.ShopLogoURL
	}
	if req.ShopBannerURL != "" {
		shopBannerURL = &req.ShopBannerURL
	}
	if req.ContactPhone != "" {
		contactPhone = &req.ContactPhone
	}
	if req.ContactEmail != "" {
		contactEmail = &req.ContactEmail
	}
	if req.Address != "" {
		address = &req.Address
	}
	if req.City != "" {
		city = &req.City
	}

	updated, err := h.merchantService.Update(r.Context(), merchant.ID, service.UpdateMerchantInput{
		ShopName:        shopName,
		ShopDescription: shopDesc,
		ShopLogoURL:     shopLogoURL,
		ShopBannerURL:   shopBannerURL,
		ContactPhone:    contactPhone,
		ContactEmail:    contactEmail,
		Address:         address,
		City:            city,

		DeliveryFeeCents:           req.DeliveryFeeCents,
		FreeDeliveryThresholdCents: req.FreeDeliveryThresholdCents,
		ClearFreeDeliveryThreshold: req.ClearFreeDeliveryThreshold,
		TIN:                        req.TIN,
		TradeLicenseNumber:         req.TradeLicenseNumber,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toMerchantProfileResponse(updated))
}

// GetShop godoc
// @Summary Get public shop information
// @Description Get public shop information by shop slug.
// @Tags shops
// @Accept json
// @Produce json
// @Param shop_slug path string true "Shop slug"
// @Success 200 {object} ShopResponse "Shop information"
// @Failure 404 {object} response.ErrorResponse "Shop not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /shops/{shop_slug} [get]
func (h *MerchantHandler) GetShop(w http.ResponseWriter, r *http.Request) {
	shopSlug, err := parseStringParam(r, "shop_slug")
	if err != nil {
		response.Error(w, err)
		return
	}

	merchant, err := h.merchantService.GetBySlug(r.Context(), shopSlug)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toShopResponse(merchant))
}

func parseStringParam(r *http.Request, param string) (string, error) {
	val := r.PathValue(param)
	if val == "" {
		val = r.URL.Query().Get(param)
	}
	if val == "" {
		return "", apperrors.BadRequest(param + " is required")
	}
	return val, nil
}

func toMerchantProfileResponse(m *domain.MerchantProfile) *MerchantProfileResponse {
	return &MerchantProfileResponse{
		ID:              m.ID.String(),
		UserID:          m.UserID.String(),
		ShopName:        m.ShopName,
		ShopSlug:        m.ShopSlug,
		ShopDescription: ptrToStr(m.ShopDescription),
		ShopLogoURL:     ptrToStr(m.ShopLogoURL),
		ShopBannerURL:   ptrToStr(m.ShopBannerURL),
		ContactPhone:    m.ContactPhone,
		ContactEmail:    ptrToStr(m.ContactEmail),
		Address:         ptrToStr(m.Address),
		City:            m.City,
		IsVerified:      m.IsVerified,
		TotalSalesCents: m.TotalSalesCents,
		CreatedAt:       m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),

		DeliveryFeeCents:           m.DeliveryFeeCents,
		FreeDeliveryThresholdCents: m.FreeDeliveryThresholdCents,
		TIN:                        m.TIN,
		TradeLicenseNumber:         m.TradeLicenseNumber,
		HasWallet:                  m.BlnkBalanceID != nil && *m.BlnkBalanceID != "",
	}
}

func toShopResponse(m *domain.MerchantProfile) *ShopResponse {
	return &ShopResponse{
		ShopSlug:     m.ShopSlug,
		ShopName:     m.ShopName,
		Description:  ptrToStr(m.ShopDescription),
		LogoURL:      ptrToStr(m.ShopLogoURL),
		BannerURL:    ptrToStr(m.ShopBannerURL),
		ContactPhone: m.ContactPhone,
		ContactEmail: ptrToStr(m.ContactEmail),
		City:         m.City,
		IsVerified:   m.IsVerified,

		DeliveryFeeCents:           m.DeliveryFeeCents,
		FreeDeliveryThresholdCents: m.FreeDeliveryThresholdCents,
	}
}

func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
