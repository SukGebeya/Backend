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
	productService   *service.ProductService
	variantService   *service.ProductVariantService
	inventoryService *service.InventoryService
	merchantService  *service.MerchantService
}

func NewProductHandler(
	productService *service.ProductService,
	variantService *service.ProductVariantService,
	inventoryService *service.InventoryService,
	merchantService *service.MerchantService,
) *ProductHandler {
	return &ProductHandler{
		productService:   productService,
		variantService:   variantService,
		inventoryService: inventoryService,
		merchantService:  merchantService,
	}
}

type CreateProductRequest struct {
	CategoryID  string `json:"category_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string `json:"name" validate:"required,min=2,max=200" example:"Traditional Ethiopian Coffee"`
	Slug        string `json:"slug" validate:"omitempty,min=2,max=200,slug" example:"traditional-ethiopian-coffee"`
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
	StockQuantity     int            `json:"stock_quantity" validate:"min=0,max=1000000" example:"25"`
}

type SetStockRequest struct {
	Quantity int `json:"quantity" validate:"min=0,max=1000000" example:"40"`
}

type InventoryResponse struct {
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
	Reserved  int    `json:"reserved"`
	Available int    `json:"available"`
	UpdatedAt string `json:"updated_at"`
}

type UpdateVariantRequest struct {
	Title             *string        `json:"title" validate:"omitempty,min=2,max=200" example:"500g Ground Coffee"`
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
	Images   []ImageResponse   `json:"images"`
	// Shop is set on the public product page so buyers see the seller and delivery fee before paying.
	Shop *ShopResponse `json:"shop,omitempty"`
}

type ProductSummaryResponse struct {
	ProductResponse
	MinPriceCents     int64   `json:"min_price_cents" example:"25000"`
	AvailableQuantity int64   `json:"available_quantity" example:"12"`
	InStock           bool    `json:"in_stock" example:"true"`
	PrimaryImageURL   *string `json:"primary_image_url,omitempty"`
	ShopName          string  `json:"shop_name" example:"Tigist Habesha Designs"`
	ShopSlug          string  `json:"shop_slug" example:"tigist-habesha"`
	MerchantVerified  bool    `json:"merchant_verified" example:"true"`
}

type ProductListResponse struct {
	Items      []ProductSummaryResponse `json:"items"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"per_page"`
	TotalPages int                      `json:"total_pages"`
}

type AddImageRequest struct {
	URL       string  `json:"url" validate:"required,url,startswith=https://,max=1000" example:"https://cdn.gebeya.example/p/kemis.jpg"`
	AltText   *string `json:"alt_text" validate:"omitempty,max=200"`
	IsPrimary bool    `json:"is_primary"`
}

type ImageResponse struct {
	ID        string  `json:"id"`
	URL       string  `json:"url"`
	AltText   *string `json:"alt_text,omitempty"`
	SortOrder int     `json:"sort_order"`
	IsPrimary bool    `json:"is_primary"`
}

func toImageResponse(img *domain.ProductImage) ImageResponse {
	return ImageResponse{ID: img.ID.String(), URL: img.URL, AltText: img.AltText, SortOrder: img.SortOrder, IsPrimary: img.IsPrimary}
}

func toImageResponses(images []domain.ProductImage) []ImageResponse {
	out := make([]ImageResponse, len(images))
	for i := range images {
		out[i] = toImageResponse(&images[i])
	}
	return out
}

func toProductSummaryResponse(p *domain.ProductSummary) ProductSummaryResponse {
	return ProductSummaryResponse{
		ProductResponse:   toProductResponse(&p.Product),
		MinPriceCents:     p.MinPriceCents,
		AvailableQuantity: p.AvailableQuantity,
		InStock:           p.AvailableQuantity > 0,
		PrimaryImageURL:   p.PrimaryImageURL,
		ShopName:          p.ShopName,
		ShopSlug:          p.ShopSlug,
		MerchantVerified:  p.MerchantVerified,
	}
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
	AvailableQuantity *int           `json:"available_quantity,omitempty"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
}

// ListProducts godoc
// @Summary Browse products
// @Description Paginated list of active products with their lowest price, current availability, primary image and shop. Search works in Amharic and English.
// @Tags products
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param q query string false "Search in product name and description"
// @Param category_id query string false "Filter by category ID"
// @Param merchant_id query string false "Filter by merchant (shop) ID"
// @Success 200 {object} ProductListResponse "List of products"
// @Failure 400 {object} response.ErrorResponse "Invalid filter"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /products [get]
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	opts := parseListOptions(r)
	query := r.URL.Query()

	filter := domain.ProductFilter{Search: query.Get("q")}
	if len([]rune(filter.Search)) > 100 {
		response.Error(w, apperrors.BadRequest("search query is too long"))
		return
	}
	for param, target := range map[string]**uuid.UUID{"category_id": &filter.CategoryID, "merchant_id": &filter.MerchantID} {
		if raw := query.Get(param); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil {
				response.Error(w, apperrors.BadRequest("invalid "+param))
				return
			}
			*target = &id
		}
	}

	result, err := h.productService.Browse(r.Context(), filter, opts)
	if err != nil {
		response.Error(w, err)
		return
	}

	items := make([]ProductSummaryResponse, len(result.Items))
	for i := range result.Items {
		items[i] = toProductSummaryResponse(&result.Items[i])
	}
	response.JSON(w, http.StatusOK, ProductListResponse{
		Items:      items,
		Total:      result.Total,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	})
}

// GetProduct godoc
// @Summary Get a product
// @Description Get an active product by ID or slug, with its variants (including how many units are available), images and shop.
// @Tags products
// @Accept json
// @Produce json
// @Param slug path string true "Product ID or slug"
// @Success 200 {object} ProductDetailResponse "Product details with variants"
// @Failure 404 {object} response.ErrorResponse "Product not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /products/{slug} [get]
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	key, err := parseStringParam(r, "slug")
	if err != nil {
		response.Error(w, err)
		return
	}

	var product *domain.Product
	if id, parseErr := uuid.Parse(key); parseErr == nil {
		product, err = h.productService.GetByID(r.Context(), id)
	} else {
		product, err = h.productService.GetBySlug(r.Context(), key)
	}
	if err != nil {
		response.Error(w, err)
		return
	}
	if product.Status != domain.ProductStatusActive {
		response.Error(w, apperrors.NotFound("product not found"))
		return
	}

	variants, err := h.variantService.GetActiveByProduct(r.Context(), product.ID)
	if err != nil {
		response.Error(w, err)
		return
	}

	merchant, err := h.merchantService.GetByID(r.Context(), product.MerchantID)
	if err != nil {
		response.Error(w, err)
		return
	}

	h.writeProductDetail(w, r, product, variants, toShopResponse(merchant))
}

// GetMyProduct godoc
// @Summary Get one of your products
// @Description Get a product you own in any status, with all its variants (including inactive ones), stock and images.
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Success 200 {object} ProductDetailResponse "Product details with variants"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Product not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /merchant/products/{id} [get]
func (h *ProductHandler) GetMyProduct(w http.ResponseWriter, r *http.Request) {
	product, ok := h.ownedProduct(w, r)
	if !ok {
		return
	}

	variants, err := h.variantService.GetByProduct(r.Context(), product.ID)
	if err != nil {
		response.Error(w, err)
		return
	}

	h.writeProductDetail(w, r, product, variants, nil)
}

func (h *ProductHandler) writeProductDetail(w http.ResponseWriter, r *http.Request, product *domain.Product, variants []domain.ProductVariant, shop *ShopResponse) {
	stock, err := h.inventoryService.GetByProductID(r.Context(), product.ID)
	if err != nil {
		response.Error(w, err)
		return
	}
	available := make(map[uuid.UUID]int, len(stock))
	for _, inv := range stock {
		available[inv.VariantID] = max(inv.Quantity-inv.Reserved, 0)
	}

	images, err := h.productService.ListImages(r.Context(), product.ID)
	if err != nil {
		response.Error(w, err)
		return
	}

	variantResponses := toVariantResponses(variants)
	for i := range variantResponses {
		qty := available[variants[i].ID]
		variantResponses[i].AvailableQuantity = &qty
	}

	response.JSON(w, http.StatusOK, ProductDetailResponse{
		Product:  toProductResponse(product),
		Variants: variantResponses,
		Images:   toImageResponses(images),
		Shop:     shop,
	})
}

// AddProductImage godoc
// @Summary Add a product image
// @Description Attach an image (an https URL from your storage or CDN) to one of your products. The first image becomes the primary one.
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Param request body AddImageRequest true "Image"
// @Success 201 {object} ImageResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Router /merchant/products/{id}/images [post]
func (h *ProductHandler) AddProductImage(w http.ResponseWriter, r *http.Request) {
	product, ok := h.ownedProduct(w, r)
	if !ok {
		return
	}

	var req AddImageRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	image, err := h.productService.AddImage(r.Context(), product.ID, req.URL, req.AltText, req.IsPrimary)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, toImageResponse(image))
}

// DeleteProductImage godoc
// @Summary Remove a product image
// @Tags merchant
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Param imageId path string true "Image ID"
// @Success 204
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /merchant/products/{id}/images/{imageId} [delete]
func (h *ProductHandler) DeleteProductImage(w http.ResponseWriter, r *http.Request) {
	product, ok := h.ownedProduct(w, r)
	if !ok {
		return
	}
	imageID, err := parseUUIDParam(r, "imageId")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.productService.DeleteImage(r.Context(), product.ID, imageID); err != nil {
		response.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) ownedProduct(w http.ResponseWriter, r *http.Request) (*domain.Product, bool) {
	productID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return nil, false
	}
	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
		return nil, false
	}
	product, err := h.productService.GetByID(r.Context(), productID)
	if err != nil {
		response.Error(w, err)
		return nil, false
	}
	if product.MerchantID != merchant.ID {
		response.Error(w, forbidden("not authorized to modify this product"))
		return nil, false
	}
	return product, true
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
// @Success 200 {object} response.PaginatedResponse "List of products"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /merchant/products [get]
func (h *ProductHandler) ListMyProducts(w http.ResponseWriter, r *http.Request) {
	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
		return
	}

	opts := parseListOptions(r)
	statusStr := r.URL.Query().Get("status")

	var result *domain.PaginatedResult[domain.Product]
	var err error
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

	products := make([]ProductResponse, len(result.Items))
	for i, p := range result.Items {
		products[i] = toProductResponse(&p)
	}

	response.JSON(w, http.StatusOK, response.PaginatedResponse{
		Items:      products,
		Total:      result.Total,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	})
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
// @Router /merchant/products [post]
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
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
// @Router /merchant/products/{id} [put]
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
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
// @Router /merchant/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
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
// @Router /merchant/products/{id}/variants [post]
func (h *ProductHandler) AddVariant(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
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
		StockQuantity:     req.StockQuantity,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, toVariantResponse(variant))
}

// SetVariantStock godoc
// @Summary Set variant stock
// @Description Set the on-hand quantity for a variant. The quantity cannot be lower than what is currently reserved by carts and unpaid orders.
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Param variantId path string true "Variant ID"
// @Param request body SetStockRequest true "Stock level"
// @Success 200 {object} InventoryResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /merchant/products/{id}/variants/{variantId}/stock [put]
func (h *ProductHandler) SetVariantStock(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	variantID, err := parseUUIDParam(r, "variantId")
	if err != nil {
		response.Error(w, err)
		return
	}

	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
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

	variant, err := h.variantService.GetByID(r.Context(), variantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	if variant.ProductID != productID {
		response.Error(w, apperrors.NotFound("variant not found"))
		return
	}

	var req SetStockRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	inv, err := h.inventoryService.SetStock(r.Context(), variantID, req.Quantity)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, InventoryResponse{
		VariantID: inv.VariantID.String(),
		Quantity:  inv.Quantity,
		Reserved:  inv.Reserved,
		Available: inv.Quantity - inv.Reserved,
		UpdatedAt: inv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// UpdateVariant godoc
// @Summary Update a variant
// @Description Update an existing product variant
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Param variantId path string true "Variant ID"
// @Param request body UpdateVariantRequest true "Variant update details"
// @Success 200 {object} VariantResponse "Updated variant"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Product or variant not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /merchant/products/{id}/variants/{variantId} [put]
func (h *ProductHandler) UpdateVariant(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	variantID, err := parseUUIDParam(r, "variantId")
	if err != nil {
		response.Error(w, err)
		return
	}

	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
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

	var req UpdateVariantRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	updated, err := h.variantService.Update(r.Context(), variantID, service.UpdateVariantInput{
		Title:             req.Title,
		PriceCents:        req.PriceCents,
		ComparePriceCents: req.ComparePriceCents,
		Attributes:        req.Attributes,
		IsActive:          req.IsActive,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toVariantResponse(updated))
}

// DeleteVariant godoc
// @Summary Delete a variant
// @Description Delete a product variant
// @Tags merchant
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Param variantId path string true "Variant ID"
// @Success 204 "No content"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Product or variant not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /merchant/products/{id}/variants/{variantId} [delete]
func (h *ProductHandler) DeleteVariant(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	variantID, err := parseUUIDParam(r, "variantId")
	if err != nil {
		response.Error(w, err)
		return
	}

	merchant, ok := middleware.GetMerchantProfile(r.Context())
	if !ok {
		response.Error(w, forbidden("merchant profile required"))
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

	if err := h.variantService.Delete(r.Context(), variantID); err != nil {
		response.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
