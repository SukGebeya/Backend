package handler

import (
	"net/http"

	"github.com/SukGebeya/Backend/internal/domain"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/SukGebeya/Backend/pkg/validation"
	"github.com/google/uuid"
)

type ProductHandler struct {
	productService  *service.ProductService
	variantService  *service.ProductVariantService
	merchantService *service.MerchantService
}

func NewProductHandler(
	productService *service.ProductService,
	variantService *service.ProductVariantService,
	merchantService *service.MerchantService,
) *ProductHandler {
	return &ProductHandler{
		productService:  productService,
		variantService:  variantService,
		merchantService: merchantService,
	}
}

type CreateProductRequest struct {
	CategoryID  string `json:"category_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string `json:"name" validate:"required,min=2,max=200" example:"Traditional Ethiopian Coffee"`
	Slug        string `json:"slug" validate:"required,min=2,max=200" example:"traditional-ethiopian-coffee"`
	Description string `json:"description" validate:"omitempty,max=2000" example:"Premium Ethiopian coffee from Yirgacheffe region"`
	Status      string `json:"status" validate:"omitempty,oneof=draft active archived" example:"draft"`
	IsDigital   bool   `json:"is_digital"`
}

type UpdateProductRequest struct {
	Name        string `json:"name" validate:"omitempty,min=2,max=200" example:"Traditional Ethiopian Coffee"`
	CategoryID  string `json:"category_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Description string `json:"description" validate:"omitempty,max=2000" example:"Premium Ethiopian coffee from Yirgacheffe region"`
	Status      string `json:"status" validate:"omitempty,oneof=draft active archived" example:"active"`
	IsDigital   *bool  `json:"is_digital"`
}

type CreateVariantRequest struct {
	SKU               string         `json:"sku" validate:"required,min=2,max=100" example:"COFFEE-YIRG-500G"`
	Title             string         `json:"title" validate:"required,min=2,max=200" example:"500g Ground Coffee"`
	PriceCents        int64          `json:"price_cents" validate:"required,min=1" example:"25000"`
	ComparePriceCents *int64         `json:"compare_price_cents" validate:"omitempty,min=1" example:"30000"`
	Currency          string         `json:"currency" validate:"required,len=3" example:"ETB"`
	Attributes        map[string]any `json:"attributes"`
	IsActive          bool           `json:"is_active"`
}

type UpdateVariantRequest struct {
	Title             string         `json:"title" validate:"omitempty,min=2,max=200" example:"500g Ground Coffee"`
	PriceCents        *int64         `json:"price_cents" validate:"omitempty,min=1" example:"25000"`
	ComparePriceCents *int64         `json:"compare_price_cents" validate:"omitempty,min=1" example:"30000"`
	Attributes        map[string]any `json:"attributes"`
	IsActive          *bool          `json:"is_active"`
}

type ProductResponse struct {
	ID          string `json:"id"`
	MerchantID  string `json:"merchant_id"`
	CategoryID  string `json:"category_id,omitempty"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	IsDigital   bool   `json:"is_digital"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ProductDetailResponse struct {
	Product  ProductResponse   `json:"product"`
	Variants []VariantResponse `json:"variants"`
}

type VariantResponse struct {
	ID                string         `json:"id"`
	ProductID         string         `json:"product_id"`
	SKU               string         `json:"sku"`
	Title             string         `json:"title"`
	PriceCents        int64          `json:"price_cents"`
	ComparePriceCents *int64         `json:"compare_price_cents,omitempty"`
	Currency          string         `json:"currency"`
	Attributes        map[string]any `json:"attributes"`
	IsActive          bool           `json:"is_active"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
}

// ListProducts godoc
// @Summary List all active products
// @Description Get a paginated list of all active products
// @Tags products
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param category_id query string false "Filter by category ID"
// @Success 200 {object} response.PaginatedResponse{items=[]ProductResponse} "List of products"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/products [get]
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	opts := parseListOptions(r)

	categoryIDStr := r.URL.Query().Get("category_id")
	if categoryIDStr != "" {
		categoryID, err := uuid.Parse(categoryIDStr)
		if err != nil {
			response.Error(w, apperrors.BadRequest("invalid category_id"))
			return
		}
		result, err := h.productService.ListByCategory(r.Context(), categoryID, opts)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.JSON(w, http.StatusOK, result)
		return
	}

	result, err := h.productService.List(r.Context(), opts)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetProduct godoc
// @Summary Get product by slug
// @Description Get a single product by its slug
// @Tags products
// @Accept json
// @Produce json
// @Param slug path string true "Product slug"
// @Success 200 {object} ProductDetailResponse "Product details with variants"
// @Failure 404 {object} response.ErrorResponse "Product not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/products/{slug} [get]
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	slug, err := parseStringParam(r, "slug")
	if err != nil {
		response.Error(w, err)
		return
	}

	product, err := h.productService.GetBySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, err)
		return
	}

	variants, err := h.variantService.GetActiveByProduct(r.Context(), product.ID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ProductDetailResponse{
		Product:  toProductResponse(product),
		Variants: toVariantResponses(variants),
	})
}

// ListMyProducts godoc
// @Summary List merchant's products
// @Description Get a paginated list of products for the authenticated merchant
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param status query string false "Filter by status (draft, active, archived)"
// @Success 200 {object} response.PaginatedResponse{items=[]ProductResponse} "List of products"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/merchant/products [get]
func (h *ProductHandler) ListMyProducts(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	merchant, err := h.merchantService.GetByUserID(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	opts := parseListOptions(r)
	statusStr := r.URL.Query().Get("status")

	var result *domain.PaginatedResult[domain.Product]
	if statusStr != "" {
		status := domain.ProductStatus(statusStr)
		if !status.IsValid() {
			response.Error(w, apperrors.BadRequest("invalid status"))
			return
		}
		result, err = h.productService.ListByMerchantAndStatus(r.Context(), merchant.ID, status, opts)
	} else {
		result, err = h.productService.ListByMerchant(r.Context(), merchant.ID, opts)
	}

	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// CreateProduct godoc
// @Summary Create a new product
// @Description Create a new product for the authenticated merchant
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateProductRequest true "Product details"
// @Success 201 {object} ProductResponse "Created product"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 409 {object} response.ErrorResponse "Product slug already exists"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/merchant/products [post]
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	merchant, err := h.merchantService.GetByUserID(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req CreateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	var categoryID *uuid.UUID
	if req.CategoryID != "" {
		cid, err := uuid.Parse(req.CategoryID)
		if err != nil {
			response.Error(w, apperrors.BadRequest("invalid category_id"))
			return
		}
		categoryID = &cid
	}

	status := domain.ProductStatusDraft
	if req.Status != "" {
		status = domain.ProductStatus(req.Status)
	}

	var description *string
	if req.Description != "" {
		description = &req.Description
	}

	product, err := h.productService.Create(r.Context(), service.CreateProductInput{
		MerchantID:  merchant.ID,
		CategoryID:  categoryID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: description,
		Status:      status,
		IsDigital:   req.IsDigital,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, toProductResponse(product))
}

// UpdateProduct godoc
// @Summary Update a product
// @Description Update an existing product
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Param request body UpdateProductRequest true "Product update details"
// @Success 200 {object} ProductResponse "Updated product"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Product not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/merchant/products/{id} [put]
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	merchant, err := h.merchantService.GetByUserID(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	product, err := h.productService.GetByID(r.Context(), productID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if product.MerchantID != merchant.ID {
		response.Error(w, forbidden("not authorized to update this product"))
		return
	}

	var req UpdateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	var name, description *string
	var categoryID *uuid.UUID
	var status *domain.ProductStatus
	var isDigital *bool

	if req.Name != "" {
		name = &req.Name
	}
	if req.CategoryID != "" {
		cid, err := uuid.Parse(req.CategoryID)
		if err != nil {
			response.Error(w, apperrors.BadRequest("invalid category_id"))
			return
		}
		categoryID = &cid
	}
	if req.Description != "" {
		description = &req.Description
	}
	if req.Status != "" {
		s := domain.ProductStatus(req.Status)
		status = &s
	}
	if req.IsDigital != nil {
		isDigital = req.IsDigital
	}

	updated, err := h.productService.Update(r.Context(), productID, service.UpdateProductInput{
		Name:        name,
		CategoryID:  categoryID,
		Description: description,
		Status:      status,
		IsDigital:   isDigital,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toProductResponse(updated))
}

// DeleteProduct godoc
// @Summary Delete a product
// @Description Delete a product and all its variants
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Success 204 "No content"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Product not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/merchant/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	merchant, err := h.merchantService.GetByUserID(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	product, err := h.productService.GetByID(r.Context(), productID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if product.MerchantID != merchant.ID {
		response.Error(w, forbidden("not authorized to delete this product"))
		return
	}

	if err := h.productService.Delete(r.Context(), productID); err != nil {
		response.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddVariant godoc
// @Summary Add a variant to a product
// @Description Add a new variant to an existing product
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Param request body CreateVariantRequest true "Variant details"
// @Success 201 {object} VariantResponse "Created variant"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Product not found"
// @Failure 409 {object} response.ErrorResponse "SKU already exists"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/merchant/products/{id}/variants [post]
func (h *ProductHandler) AddVariant(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	merchant, err := h.merchantService.GetByUserID(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	product, err := h.productService.GetByID(r.Context(), productID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if product.MerchantID != merchant.ID {
		response.Error(w, forbidden("not authorized to modify this product"))
		return
	}

	var req CreateVariantRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "ETB"
	}

	variant, err := h.variantService.Create(r.Context(), service.CreateVariantInput{
		ProductID:         productID,
		SKU:               req.SKU,
		Title:             req.Title,
		PriceCents:        req.PriceCents,
		ComparePriceCents: req.ComparePriceCents,
		Currency:          currency,
		Attributes:        req.Attributes,
		IsActive:          req.IsActive,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, toVariantResponse(variant))
}

func toProductResponse(p *domain.Product) ProductResponse {
	var categoryID string
	if p.CategoryID != nil {
		categoryID = p.CategoryID.String()
	}

	var description string
	if p.Description != nil {
		description = *p.Description
	}

	return ProductResponse{
		ID:          p.ID.String(),
		MerchantID:  p.MerchantID.String(),
		CategoryID:  categoryID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: description,
		Status:      string(p.Status),
		IsDigital:   p.IsDigital,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toVariantResponse(v *domain.ProductVariant) VariantResponse {
	return VariantResponse{
		ID:                v.ID.String(),
		ProductID:         v.ProductID.String(),
		SKU:               v.SKU,
		Title:             v.Title,
		PriceCents:        v.PriceCents,
		ComparePriceCents: v.ComparePriceCents,
		Currency:          v.Currency,
		Attributes:        v.Attributes,
		IsActive:          v.IsActive,
		CreatedAt:         v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:         v.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toVariantResponses(variants []domain.ProductVariant) []VariantResponse {
	responses := make([]VariantResponse, len(variants))
	for i, v := range variants {
		responses[i] = toVariantResponse(&v)
	}
	return responses
}
