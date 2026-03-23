package handler

import (
	"net/http"

	"github.com/SukGebeya/Backend/internal/domain"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/SukGebeya/Backend/pkg/validation"
	"github.com/google/uuid"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
}

func NewCategoryHandler(categoryService *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

type CreateCategoryRequest struct {
	ParentID  string `json:"parent_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string `json:"name" validate:"required,min=2,max=100" example:"Clothing & Fashion"`
	Slug      string `json:"slug" validate:"required,min=2,max=100,slug" example:"clothing-fashion"`
	IconURL   string `json:"icon_url" validate:"omitempty,url" example:"https://example.com/icons/clothing.svg"`
	IsActive  bool   `json:"is_active"`
	SortOrder int    `json:"sort_order" example:"1"`
}

type UpdateCategoryRequest struct {
	Name      string `json:"name" validate:"omitempty,min=2,max=100" example:"Clothing & Fashion"`
	Slug      string `json:"slug" validate:"omitempty,min=2,max=100,slug" example:"clothing-fashion"`
	IconURL   string `json:"icon_url" validate:"omitempty,url" example:"https://example.com/icons/clothing.svg"`
	IsActive  *bool  `json:"is_active"`
	SortOrder *int   `json:"sort_order" example:"1"`
}

type CategoryResponse struct {
	ID        string `json:"id"`
	ParentID  string `json:"parent_id,omitempty"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	IconURL   string `json:"icon_url,omitempty"`
	IsActive  bool   `json:"is_active"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
}

// ListCategories godoc
// @Summary List all active categories
// @Description Get a list of all active categories
// @Tags categories
// @Accept json
// @Produce json
// @Success 200 {array} CategoryResponse "List of categories"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/categories [get]
func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categoryService.List(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}

	responses := make([]CategoryResponse, len(categories))
	for i, c := range categories {
		responses[i] = toCategoryResponse(&c)
	}

	response.JSON(w, http.StatusOK, responses)
}

// GetCategory godoc
// @Summary Get category by ID
// @Description Get a single category by its ID
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} CategoryResponse "Category details"
// @Failure 404 {object} response.ErrorResponse "Category not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/categories/{id} [get]
func (h *CategoryHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	category, err := h.categoryService.GetByID(r.Context(), categoryID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toCategoryResponse(category))
}

// GetCategoryBySlug godoc
// @Summary Get category by slug
// @Description Get a single category by its slug
// @Tags categories
// @Accept json
// @Produce json
// @Param slug path string true "Category slug"
// @Success 200 {object} CategoryResponse "Category details"
// @Failure 404 {object} response.ErrorResponse "Category not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/categories/slug/{slug} [get]
func (h *CategoryHandler) GetCategoryBySlug(w http.ResponseWriter, r *http.Request) {
	slug, err := parseStringParam(r, "slug")
	if err != nil {
		response.Error(w, err)
		return
	}

	category, err := h.categoryService.GetBySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toCategoryResponse(category))
}

// CreateCategory godoc
// @Summary Create a new category
// @Description Create a new category (admin only)
// @Tags categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateCategoryRequest true "Category details"
// @Success 201 {object} CategoryResponse "Created category"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 409 {object} response.ErrorResponse "Category slug already exists"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/admin/categories [post]
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	var parentID *string
	if req.ParentID != "" {
		parentID = &req.ParentID
	}

	var iconURL *string
	if req.IconURL != "" {
		iconURL = &req.IconURL
	}

	category, err := h.categoryService.Create(r.Context(), service.CreateCategoryInput{
		ParentID:  parentID,
		Name:      req.Name,
		Slug:      req.Slug,
		IconURL:   iconURL,
		IsActive:  req.IsActive,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, toCategoryResponse(category))
}

// UpdateCategory godoc
// @Summary Update a category
// @Description Update an existing category (admin only)
// @Tags categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Category ID"
// @Param request body UpdateCategoryRequest true "Category update details"
// @Success 200 {object} CategoryResponse "Updated category"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Category not found"
// @Failure 409 {object} response.ErrorResponse "Category slug already exists"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/admin/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	var req UpdateCategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	var name, slug *string
	var iconURL *string
	var isActive *bool
	var sortOrder *int

	if req.Name != "" {
		name = &req.Name
	}
	if req.Slug != "" {
		slug = &req.Slug
	}
	if req.IconURL != "" {
		iconURL = &req.IconURL
	}
	if req.IsActive != nil {
		isActive = req.IsActive
	}
	if req.SortOrder != nil {
		sortOrder = req.SortOrder
	}

	category, err := h.categoryService.Update(r.Context(), categoryID, service.UpdateCategoryInput{
		Name:      name,
		Slug:      slug,
		IconURL:   iconURL,
		IsActive:  isActive,
		SortOrder: sortOrder,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toCategoryResponse(category))
}

// DeleteCategory godoc
// @Summary Delete a category
// @Description Delete a category (admin only)
// @Tags categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Category ID"
// @Success 204 "No content"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Category not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/admin/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	if err := h.categoryService.Delete(r.Context(), categoryID); err != nil {
		response.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toCategoryResponse(c *domain.Category) CategoryResponse {
	var parentID string
	if c.ParentID != nil {
		parentID = c.ParentID.String()
	}

	var iconURL string
	if c.IconURL != nil {
		iconURL = *c.IconURL
	}

	return CategoryResponse{
		ID:        c.ID.String(),
		ParentID:  parentID,
		Name:      c.Name,
		Slug:      c.Slug,
		IconURL:   iconURL,
		IsActive:  c.IsActive,
		SortOrder: c.SortOrder,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func parseUUIDParamFromQuery(r *http.Request, param string) (uuid.UUID, error) {
	val := r.URL.Query().Get(param)
	if val == "" {
		return uuid.UUID{}, apperrors.BadRequest(param + " is required")
	}

	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.UUID{}, apperrors.BadRequest("invalid " + param)
	}

	return id, nil
}
