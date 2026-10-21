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

type AddressHandler struct {
	addressService *service.AddressService
}

func NewAddressHandler(addressService *service.AddressService) *AddressHandler {
	return &AddressHandler{
		addressService: addressService,
	}
}

type CreateAddressRequest struct {
	FullName  string  `json:"full_name" validate:"required,min=2,max=100" example:"Abebe Bikila"`
	Phone     string  `json:"phone" validate:"required,etphone" example:"0911234567"`
	Line1     string  `json:"line1" validate:"required,min=5,max=200" example:"Bole Sub-city, Woreda 03"`
	Line2     *string `json:"line2" validate:"omitempty,max=200" example:"Apartment 4B"`
	City      string  `json:"city" validate:"required,min=2,max=100" example:"Addis Ababa"`
	Subcity   *string `json:"subcity" validate:"omitempty,max=100" example:"Bole"`
	Woreda    *string `json:"woreda" validate:"omitempty,max=50" example:"03"`
	IsDefault bool    `json:"is_default" example:"false"`
}

type UpdateAddressRequest struct {
	FullName *string `json:"full_name" validate:"omitempty,min=2,max=100" example:"Abebe Bikila"`
	Phone    *string `json:"phone" validate:"omitempty,etphone" example:"0911234567"`
	Line1    *string `json:"line1" validate:"omitempty,min=5,max=200" example:"Bole Sub-city, Woreda 03"`
	Line2    *string `json:"line2" validate:"omitempty,max=200" example:"Apartment 4B"`
	City     *string `json:"city" validate:"omitempty,min=2,max=100" example:"Addis Ababa"`
	Subcity  *string `json:"subcity" validate:"omitempty,max=100" example:"Bole"`
	Woreda   *string `json:"woreda" validate:"omitempty,max=50" example:"03"`
}

type AddressResponse struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	FullName  string  `json:"full_name"`
	Phone     string  `json:"phone"`
	Line1     string  `json:"line1"`
	Line2     *string `json:"line2,omitempty"`
	City      string  `json:"city"`
	Subcity   *string `json:"subcity,omitempty"`
	Woreda    *string `json:"woreda,omitempty"`
	IsDefault bool    `json:"is_default"`
	CreatedAt string  `json:"created_at"`
}

// ListAddresses godoc
// @Summary List user's addresses
// @Description Get all addresses for the authenticated user
// @Tags addresses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} AddressResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /addresses [get]
func (h *AddressHandler) ListAddresses(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	addresses, err := h.addressService.ListByUserID(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	result := make([]AddressResponse, len(addresses))
	for i, addr := range addresses {
		result[i] = toAddressResponse(&addr)
	}

	response.JSON(w, http.StatusOK, result)
}

// GetAddress godoc
// @Summary Get address by ID
// @Description Get a specific address by ID
// @Tags addresses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Address ID"
// @Success 200 {object} AddressResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /addresses/{id} [get]
func (h *AddressHandler) GetAddress(w http.ResponseWriter, r *http.Request) {
	addressID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	address, err := h.addressService.GetByID(r.Context(), addressID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if address.UserID != userID {
		response.Error(w, forbidden("not authorized to access this address"))
		return
	}

	response.JSON(w, http.StatusOK, toAddressResponse(address))
}

// CreateAddress godoc
// @Summary Create a new address
// @Description Create a new address for the authenticated user
// @Tags addresses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateAddressRequest true "Address details"
// @Success 201 {object} AddressResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /addresses [post]
func (h *AddressHandler) CreateAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	var req CreateAddressRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	city := req.City
	if city == "" {
		city = "Addis Ababa"
	}

	address, err := h.addressService.Create(r.Context(), service.CreateAddressInput{
		UserID:    userID,
		FullName:  req.FullName,
		Phone:     req.Phone,
		Line1:     req.Line1,
		Line2:     req.Line2,
		City:      city,
		Subcity:   req.Subcity,
		Woreda:    req.Woreda,
		IsDefault: req.IsDefault,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, toAddressResponse(address))
}

// UpdateAddress godoc
// @Summary Update an address
// @Description Update an existing address
// @Tags addresses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Address ID"
// @Param request body UpdateAddressRequest true "Address update details"
// @Success 200 {object} AddressResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /addresses/{id} [put]
func (h *AddressHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	addressID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	address, err := h.addressService.GetByID(r.Context(), addressID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if address.UserID != userID {
		response.Error(w, forbidden("not authorized to update this address"))
		return
	}

	var req UpdateAddressRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	updated, err := h.addressService.Update(r.Context(), addressID, service.UpdateAddressInput{
		FullName: req.FullName,
		Phone:    req.Phone,
		Line1:    req.Line1,
		Line2:    req.Line2,
		City:     req.City,
		Subcity:  req.Subcity,
		Woreda:   req.Woreda,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toAddressResponse(updated))
}

// SetDefaultAddress godoc
// @Summary Set address as default
// @Description Set an address as the default address for the user
// @Tags addresses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Address ID"
// @Success 200 {object} AddressResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /addresses/{id}/default [put]
func (h *AddressHandler) SetDefaultAddress(w http.ResponseWriter, r *http.Request) {
	addressID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	address, err := h.addressService.GetByID(r.Context(), addressID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if address.UserID != userID {
		response.Error(w, forbidden("not authorized to modify this address"))
		return
	}

	updated, err := h.addressService.SetDefault(r.Context(), userID, addressID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toAddressResponse(updated))
}

// DeleteAddress godoc
// @Summary Delete an address
// @Description Delete an address
// @Tags addresses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Address ID"
// @Success 204 "No content"
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /addresses/{id} [delete]
func (h *AddressHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	addressID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}

	address, err := h.addressService.GetByID(r.Context(), addressID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if address.UserID != userID {
		response.Error(w, forbidden("not authorized to delete this address"))
		return
	}

	if err := h.addressService.Delete(r.Context(), addressID); err != nil {
		response.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toAddressResponse(a *domain.Address) AddressResponse {
	return AddressResponse{
		ID:        a.ID.String(),
		UserID:    a.UserID.String(),
		FullName:  a.FullName,
		Phone:     a.Phone,
		Line1:     a.Line1,
		Line2:     a.Line2,
		City:      a.City,
		Subcity:   a.Subcity,
		Woreda:    a.Woreda,
		IsDefault: a.IsDefault,
		CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
